package httpapi

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

const sessionCookieName = "snakerule_session"

type AuthConfig struct {
	Email      string
	Name       string
	Password   string
	SessionTTL time.Duration
}

type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]time.Time
	ttl      time.Duration
}

func newSessionStore(ttl time.Duration) *sessionStore {
	return &sessionStore{sessions: make(map[string]time.Time), ttl: ttl}
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeBody(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	userOK := subtle.ConstantTimeCompare([]byte(strings.TrimSpace(input.Email)), []byte(s.auth.Email))
	passOK := subtle.ConstantTimeCompare([]byte(input.Password), []byte(s.auth.Password))
	if userOK != 1 || passOK != 1 {
		writeError(w, http.StatusUnauthorized, errors.New("邮箱或密码错误"))
		return
	}

	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		writeError(w, http.StatusInternalServerError, errors.New("unable to create session"))
		return
	}
	token := base64.RawURLEncoding.EncodeToString(bytes)
	expires := time.Now().Add(s.sessions.ttl)
	s.sessions.mu.Lock()
	s.sessions.sessions[token] = expires
	s.sessions.mu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: token, Path: "/", HttpOnly: true,
		Secure:   r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https"),
		SameSite: http.SameSiteLaxMode, Expires: expires, MaxAge: int(s.sessions.ttl.Seconds()),
	})
	writeJSON(w, http.StatusOK, s.sessionResponse())
}

func (s *Server) currentSession(w http.ResponseWriter, r *http.Request) {
	if !s.validSession(r) {
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
		return
	}
	writeJSON(w, http.StatusOK, s.sessionResponse())
}

func (s *Server) sessionResponse() map[string]any {
	return map[string]any{"authenticated": true, "user": map[string]any{"email": s.auth.Email, "name": s.auth.Name, "is_superadmin": true}}
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		s.sessions.mu.Lock()
		delete(s.sessions.sessions, cookie.Value)
		s.sessions.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0), Secure: r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")})
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": false})
}

func (s *Server) validSession(r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	now := time.Now()
	s.sessions.mu.Lock()
	defer s.sessions.mu.Unlock()
	expires, ok := s.sessions.sessions[cookie.Value]
	if !ok {
		return false
	}
	if now.After(expires) {
		delete(s.sessions.sessions, cookie.Value)
		return false
	}
	return true
}

func (s *Server) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.validSession(r) {
			writeError(w, http.StatusUnauthorized, errors.New("authentication required"))
			return
		}
		next.ServeHTTP(w, r)
	})
}
