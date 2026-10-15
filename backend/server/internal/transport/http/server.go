package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/szplanner/snakerule/internal/domain"
	"github.com/szplanner/snakerule/internal/repo"
	"github.com/szplanner/snakerule/internal/service"
	appversion "github.com/szplanner/snakerule/internal/version"
)

type Server struct {
	rulesets *service.Rulesets
	engine   *service.Engine
	logger   *slog.Logger
	auth     AuthConfig
	sessions *sessionStore
}

func New(rulesets *service.Rulesets, engine *service.Engine, logger *slog.Logger, auth AuthConfig) *Server {
	return &Server{rulesets: rulesets, engine: engine, logger: logger, auth: auth, sessions: newSessionStore(auth.SessionTTL)}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.index)
	mux.Handle("GET /assets/", s.assets())
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /api/v1/meta", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"version": appversion.Version})
	})
	mux.HandleFunc("POST /api/v1/auth/login", s.login)
	mux.HandleFunc("GET /api/v1/auth/session", s.currentSession)
	mux.HandleFunc("POST /api/v1/auth/logout", s.logout)
	mux.Handle("GET /api/v1/rulesets", s.requireSession(http.HandlerFunc(s.listRulesets)))
	mux.Handle("POST /api/v1/rulesets", s.requireSession(http.HandlerFunc(s.saveRuleset)))
	mux.Handle("GET /api/v1/rulesets/{id}", s.requireSession(http.HandlerFunc(s.getRuleset)))
	mux.Handle("PUT /api/v1/rulesets/{id}", s.requireSession(http.HandlerFunc(s.saveRuleset)))
	mux.Handle("DELETE /api/v1/rulesets/{id}", s.requireSession(http.HandlerFunc(s.deleteRuleset)))
	mux.Handle("POST /api/v1/engine/evaluate", s.requireSession(http.HandlerFunc(s.evaluate)))
	return s.recover(s.accessLog(mux))
}

func (s *Server) listRulesets(w http.ResponseWriter, r *http.Request) {
	items, err := s.rulesets.List(r.Context())
	if err != nil {
		writeError(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
func (s *Server) getRuleset(w http.ResponseWriter, r *http.Request) {
	item, err := s.rulesets.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	writeJSON(w, 200, item)
}
func (s *Server) deleteRuleset(w http.ResponseWriter, r *http.Request) {
	if err := s.rulesets.Delete(r.Context(), r.PathValue("id")); err != nil {
		s.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (s *Server) saveRuleset(w http.ResponseWriter, r *http.Request) {
	var item domain.Ruleset
	if err := decodeBody(w, r, &item); err != nil {
		writeError(w, 400, err)
		return
	}
	if id := r.PathValue("id"); id != "" && item.ID != id {
		writeError(w, 400, errors.New("path id must match ruleset id"))
		return
	}
	if err := s.rulesets.Save(r.Context(), item); err != nil {
		s.writeServiceError(w, err)
		return
	}
	w.Header().Set("Location", "/api/v1/rulesets/"+item.ID)
	writeJSON(w, 200, item)
}
func (s *Server) evaluate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RulesetID string         `json:"rulesetId"`
		Facts     map[string]any `json:"facts"`
	}
	if err := decodeBody(w, r, &req); err != nil {
		writeError(w, 400, err)
		return
	}
	if strings.TrimSpace(req.RulesetID) == "" {
		writeError(w, 400, errors.New("rulesetId is required"))
		return
	}
	decision, err := s.engine.Evaluate(r.Context(), req.RulesetID, req.Facts)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	writeJSON(w, 200, decision)
}
func (s *Server) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repo.ErrNotFound):
		writeError(w, 404, err)
	case errors.Is(err, service.ErrInvalid):
		writeError(w, 400, err)
	default:
		writeError(w, 500, err)
	}
}

func decodeBody(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON value")
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"message": err.Error()}})
}
func (s *Server) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.Error("HTTP handler panic", "panic", recovered)
				writeError(w, 500, errors.New("internal server error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
func (s *Server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Info("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}
