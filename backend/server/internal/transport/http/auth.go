package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/szplanner/snakerule/internal/domain"
	"github.com/szplanner/snakerule/internal/repo"
)

const sessionCookieName = "snakerule_session"

type projectKeyAuthContextKey struct{}
type invocationCredentialContextKey struct{}

type invocationCredential struct {
	ID   string
	Type string
}

func invocationCredentialFrom(ctx context.Context) invocationCredential {
	credential, _ := ctx.Value(invocationCredentialContextKey{}).(invocationCredential)
	if credential.Type == "" {
		credential.Type = "session"
	}
	return credential
}

func isProjectKeyRequest(ctx context.Context) bool {
	value, _ := ctx.Value(projectKeyAuthContextKey{}).(bool)
	return value
}

type AuthConfig struct {
	Email            string
	Name             string
	Password         string
	SessionTTL       time.Duration
	SessionStorePath string
	JWTSigningKey    []byte
}

type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]session
	ttl      time.Duration
	path     string
}

type session struct {
	ExpiresAt   time.Time `json:"expires_at"`
	WorkspaceID string    `json:"workspace_id"`
	ProjectID   string    `json:"project_id"`
}

func newSessionStore(ttl time.Duration, path string) *sessionStore {
	s := &sessionStore{sessions: make(map[string]session), ttl: ttl, path: path}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &s.sessions)
	}
	return s
}

func (s *sessionStore) persist() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.sessions, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(s.path+".tmp", b, 0600); err != nil {
		return err
	}
	return os.Rename(s.path+".tmp", s.path)
}

type projectJWTClaims struct {
	Issuer    string `json:"iss"`
	Subject   string `json:"sub"`
	Workspace string `json:"workspace"`
	Project   string `json:"project"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

func (s *Server) authenticateProjectJWT(token string) (domain.APIKeyCredential, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || len(s.jwtSigningKey) < 32 {
		return domain.APIKeyCredential{}, errors.New("invalid bearer token")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return domain.APIKeyCredential{}, errors.New("invalid bearer token")
	}
	var header struct {
		Algorithm string `json:"alg"`
	}
	if json.Unmarshal(headerBytes, &header) != nil || header.Algorithm != "HS256" {
		return domain.APIKeyCredential{}, errors.New("invalid bearer token algorithm")
	}
	mac := hmac.New(sha256.New, s.jwtSigningKey)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(signature, mac.Sum(nil)) {
		return domain.APIKeyCredential{}, errors.New("invalid bearer token signature")
	}
	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return domain.APIKeyCredential{}, errors.New("invalid bearer token claims")
	}
	var claims projectJWTClaims
	if json.Unmarshal(claimsBytes, &claims) != nil || claims.Issuer != "snakerule" || claims.Subject == "" || claims.Workspace == "" || claims.Project == "" || claims.ExpiresAt <= time.Now().Unix() || claims.IssuedAt > time.Now().Add(time.Minute).Unix() {
		return domain.APIKeyCredential{}, errors.New("expired or invalid bearer token")
	}
	digest := sha256.Sum256([]byte(token))
	credential, err := s.projectKeys.FindByHash(context.Background(), base64.RawURLEncoding.EncodeToString(digest[:]))
	if err != nil || credential.Type != "jwt" || credential.ID != claims.Subject || credential.Workspace != claims.Workspace || credential.Project != claims.Project {
		return domain.APIKeyCredential{}, errors.New("revoked or invalid bearer token")
	}
	return credential, nil
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
	now := time.Now()
	expires := now.Add(s.sessions.ttl)
	s.sessions.mu.Lock()
	for existingToken, existingSession := range s.sessions.sessions {
		if !existingSession.ExpiresAt.After(now) {
			delete(s.sessions.sessions, existingToken)
		}
	}
	s.sessions.sessions[token] = session{ExpiresAt: expires, WorkspaceID: repo.DefaultWorkspaceID, ProjectID: repo.DefaultProjectID}
	if err := s.sessions.persist(); err != nil {
		s.sessions.mu.Unlock()
		writeError(w, http.StatusInternalServerError, errors.New("unable to persist session"))
		return
	}
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
	current, ok := s.sessions.sessions[cookie.Value]
	if !ok {
		return false
	}
	if now.After(current.ExpiresAt) {
		delete(s.sessions.sessions, cookie.Value)
		_ = s.sessions.persist()
		return false
	}
	return true
}

func (s *sessionStore) workspaceID(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return repo.DefaultWorkspaceID
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.sessions[cookie.Value]
	if !ok || time.Now().After(current.ExpiresAt) {
		return repo.DefaultWorkspaceID
	}
	if current.WorkspaceID == "" {
		return repo.DefaultWorkspaceID
	}
	return current.WorkspaceID
}

func (s *sessionStore) setWorkspace(r *http.Request, workspaceID string) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.sessions[cookie.Value]
	if !ok || time.Now().After(current.ExpiresAt) {
		return false
	}
	current.WorkspaceID = workspaceID
	current.ProjectID = repo.DefaultProjectID
	s.sessions[cookie.Value] = current
	if s.persist() != nil {
		return false
	}
	return true
}

func (s *sessionStore) projectID(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return repo.DefaultProjectID
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.sessions[cookie.Value]
	if !ok || time.Now().After(current.ExpiresAt) || current.ProjectID == "" {
		return repo.DefaultProjectID
	}
	return current.ProjectID
}

func (s *sessionStore) setProject(r *http.Request, projectID string) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.sessions[cookie.Value]
	if !ok || time.Now().After(current.ExpiresAt) {
		return false
	}
	current.ProjectID = projectID
	s.sessions[cookie.Value] = current
	if s.persist() != nil {
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
		ctx := repo.WithProject(repo.WithWorkspace(r.Context(), s.sessions.workspaceID(r)), s.sessions.projectID(r))
		ctx = context.WithValue(ctx, invocationCredentialContextKey{}, invocationCredential{Type: "session"})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) requireSessionOrProjectKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.validSession(r) {
			ctx := repo.WithProject(repo.WithWorkspace(r.Context(), s.sessions.workspaceID(r)), s.sessions.projectID(r))
			ctx = context.WithValue(ctx, invocationCredentialContextKey{}, invocationCredential{Type: "session"})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		var credential domain.APIKeyCredential
		var err error
		if rawKey := strings.TrimSpace(r.Header.Get("X-API-Key")); rawKey != "" {
			hash := sha256.Sum256([]byte(rawKey))
			credential, err = s.projectKeys.FindByHash(r.Context(), base64.RawURLEncoding.EncodeToString(hash[:]))
			if err == nil && credential.Type == "" {
				credential.Type = "apikey"
			}
			if err == nil && credential.Type != "" && credential.Type != "apikey" {
				err = errors.New("credential type does not match X-API-Key")
			}
		} else if authz := strings.TrimSpace(r.Header.Get("Authorization")); strings.HasPrefix(strings.ToLower(authz), "bearer ") {
			credential, err = s.authenticateProjectJWT(strings.TrimSpace(authz[len("Bearer "):]))
		} else {
			err = errors.New("authentication required")
		}
		if err != nil {
			writeError(w, http.StatusUnauthorized, errors.New("invalid project API key or JWT"))
			return
		}
		ctx := repo.WithProject(repo.WithWorkspace(r.Context(), credential.Workspace), credential.Project)
		ctx = context.WithValue(ctx, projectKeyAuthContextKey{}, true)
		ctx = context.WithValue(ctx, invocationCredentialContextKey{}, invocationCredential{ID: credential.ID, Type: credential.Type})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
