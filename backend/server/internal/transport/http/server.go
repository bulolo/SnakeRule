package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/szplanner/snakerule/internal/licensing"
	"github.com/szplanner/snakerule/internal/repo"
	"github.com/szplanner/snakerule/internal/service"
)

type Server struct {
	rulesets         *service.RulesetService
	engine           *service.Engine
	logger           *slog.Logger
	auth             AuthConfig
	sessions         *sessionStore
	license          *licensing.Manager
	organizations    *service.OrganizationService
	alertChannels    repo.AlertChannelRepository
	platformSettings repo.PlatformSettingsRepository
	projects         repo.ProjectRepository
	projectKeys      repo.ProjectKeyRepository
	users            *repo.PersistentUserRepository
	notifications    *repo.PersistentNotificationRepository
	invocationLogs   *repo.PersistentInvocationLogRepository
	jwtSigningKey    []byte
}

func NewServer(rulesets *service.RulesetService, engine *service.Engine, logger *slog.Logger, auth AuthConfig, license *licensing.Manager, organizations *service.OrganizationService) *Server {
	return &Server{rulesets: rulesets, engine: engine, logger: logger, auth: auth, sessions: newSessionStore(auth.SessionTTL, auth.SessionStoreKey), license: license, organizations: organizations, jwtSigningKey: auth.JWTSigningKey}
}

func (s *Server) SetAlertChannelRepository(repository repo.AlertChannelRepository) {
	s.alertChannels = repository
}

func (s *Server) SetPlatformSettingsRepository(repository repo.PlatformSettingsRepository) {
	s.platformSettings = repository
}

func (s *Server) SetProjectRepositories(projects repo.ProjectRepository, keys repo.ProjectKeyRepository) {
	s.projects, s.projectKeys = projects, keys
}

func (s *Server) SetUserRepository(users *repo.PersistentUserRepository) { s.users = users }

func (s *Server) SetNotificationRepository(repository *repo.PersistentNotificationRepository) {
	s.notifications = repository
}

func (s *Server) SetInvocationLogRepository(repository *repo.PersistentInvocationLogRepository) {
	s.invocationLogs = repository
}

func (s *Server) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repo.ErrNotFound):
		writeError(w, 404, err)
	case errors.Is(err, service.ErrInvalid):
		writeError(w, 400, err)
	default:
		s.logger.Error("request failed", "error", err)
		writeError(w, 500, err)
	}
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
