package httpapi

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
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
	ExpiresAt       time.Time `json:"expires_at"`
	OrganizationID  string    `json:"organization_id"`
	ProjectID       string    `json:"project_id"`
	Email           string    `json:"email"`
	Name            string    `json:"name"`
	Role            string    `json:"role"`
	OrganizationIDs []string  `json:"organization_ids,omitempty"`
}

func newSessionStore(ttl time.Duration, path string) *sessionStore {
	s := &sessionStore{sessions: make(map[string]session), ttl: ttl, path: path}
	_ = repo.LoadPersistent(path, &s.sessions)
	return s
}

func (s *sessionStore) persist() error {
	return repo.SavePersistent(s.path, s.sessions)
}

type projectJWTClaims struct {
	Issuer       string `json:"iss"`
	Subject      string `json:"sub"`
	Organization string `json:"organization"`
	Project      string `json:"project"`
	IssuedAt     int64  `json:"iat"`
	ExpiresAt    int64  `json:"exp"`
}

func encryptProjectSigningSecret(master []byte, secret string) (string, error) {
	block, err := aes.NewCipher(master[:32])
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(append(nonce, aead.Seal(nil, nonce, []byte(secret), nil)...)), nil
}

func decryptProjectSigningSecret(master []byte, encoded string) (string, error) {
	block, err := aes.NewCipher(master[:32])
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(data) < aead.NonceSize() {
		return "", errors.New("invalid signing key")
	}
	nonce, ciphertext := data[:aead.NonceSize()], data[aead.NonceSize():]
	plain, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
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
		KeyID     string `json:"kid"`
	}
	if json.Unmarshal(headerBytes, &header) != nil || header.Algorithm != "HS256" {
		return domain.APIKeyCredential{}, errors.New("invalid bearer token algorithm")
	}
	var credential domain.APIKeyCredential
	var signingKey []byte
	if header.KeyID != "" {
		credential, err = s.projectKeys.FindByID(context.Background(), header.KeyID)
		if err != nil || credential.Type != "jwt" || credential.CredentialKind != "signing_key" {
			return domain.APIKeyCredential{}, errors.New("invalid signing key")
		}
		if credential.ExpiresAt != nil && !credential.ExpiresAt.After(time.Now()) {
			return domain.APIKeyCredential{}, errors.New("expired signing key")
		}
		secret, decryptErr := decryptProjectSigningSecret(s.jwtSigningKey, credential.SigningSecretEncrypted)
		if decryptErr != nil {
			return domain.APIKeyCredential{}, errors.New("invalid signing key")
		}
		signingKey = []byte(secret)
	} else {
		signingKey = s.jwtSigningKey
	}
	mac := hmac.New(sha256.New, signingKey)
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
	if json.Unmarshal(claimsBytes, &claims) != nil || claims.ExpiresAt <= time.Now().Unix() || claims.IssuedAt > time.Now().Add(time.Minute).Unix() {
		return domain.APIKeyCredential{}, errors.New("expired or invalid bearer token")
	}
	if header.KeyID != "" {
		return credential, nil
	}
	if claims.Issuer != "snakerule" || claims.Subject == "" || claims.Organization == "" || claims.Project == "" {
		return domain.APIKeyCredential{}, errors.New("expired or invalid bearer token")
	}
	digest := sha256.Sum256([]byte(token))
	credential, err = s.projectKeys.FindByHash(context.Background(), base64.RawURLEncoding.EncodeToString(digest[:]))
	if err != nil || credential.Type != "jwt" || credential.ID != claims.Subject || credential.Organization != claims.Organization || credential.Project != claims.Project {
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
	user := domain.User{Email: s.auth.Email, Name: s.auth.Name, Role: "superadmin", OrganizationIDs: []string{repo.DefaultOrganizationID}}
	valid := false
	if s.users != nil {
		if found, ok := s.users.Verify(input.Email, input.Password); ok {
			user = found
			valid = true
		}
	} else {
		valid = subtle.ConstantTimeCompare([]byte(strings.ToLower(strings.TrimSpace(input.Email))), []byte(strings.ToLower(s.auth.Email))) == 1 && subtle.ConstantTimeCompare([]byte(input.Password), []byte(s.auth.Password)) == 1
	}
	if !valid {
		writeError(w, http.StatusUnauthorized, errors.New("邮箱或密码错误"))
		return
	}
	organizationID := repo.DefaultOrganizationID
	if len(user.OrganizationIDs) > 0 {
		organizationID = user.OrganizationIDs[0]
	} else if user.Role != "superadmin" {
		writeError(w, http.StatusForbidden, errors.New("账号尚未分配组织，请联系超级管理员"))
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
	s.sessions.sessions[token] = session{ExpiresAt: expires, OrganizationID: organizationID, Email: user.Email, Name: user.Name, Role: user.Role, OrganizationIDs: user.OrganizationIDs}
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
	writeJSON(w, http.StatusOK, s.sessionResponse(r))
}

func (s *Server) currentSession(w http.ResponseWriter, r *http.Request) {
	if !s.authenticatedSession(r) {
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
		return
	}
	writeJSON(w, http.StatusOK, s.sessionResponse(r))
}

func (s *Server) sessionResponse(r *http.Request) map[string]any {
	current := s.sessionFromRequest(r)
	role := current.Role
	if s.users != nil {
		if user, ok := s.users.FindByEmail(current.Email); ok && user.OrganizationRoles[current.OrganizationID] != "" {
			role = user.OrganizationRoles[current.OrganizationID]
		}
	}
	return map[string]any{"authenticated": true, "user": map[string]any{"email": current.Email, "name": current.Name, "role": role, "is_superadmin": current.Role == "superadmin"}}
}

func (s *Server) sessionFromRequest(r *http.Request) session {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return session{}
	}
	s.sessions.mu.Lock()
	defer s.sessions.mu.Unlock()
	current := s.sessions.sessions[cookie.Value]
	if current.Role == "" && (current.Email == "" || strings.EqualFold(current.Email, s.auth.Email)) {
		current.Role = "superadmin"
		current.Email = s.auth.Email
		current.Name = s.auth.Name
	}
	if s.users != nil && current.Email != "" {
		if user, ok := s.users.FindByEmail(current.Email); ok {
			current.Role = user.Role
			current.Name = user.Name
			current.OrganizationIDs = user.OrganizationIDs
		}
	}
	return current
}
func (s *Server) isSuperAdmin(r *http.Request) bool {
	current := s.sessionFromRequest(r)
	return current.Role == "superadmin"
}
func (s *Server) canAccessOrganization(r *http.Request, organizationID string) bool {
	if s.isSuperAdmin(r) {
		return true
	}
	for _, id := range s.sessionFromRequest(r).OrganizationIDs {
		if id == organizationID {
			return true
		}
	}
	return false
}

func (s *Server) canManageOrganization(r *http.Request, organizationID string) bool {
	current := s.sessionFromRequest(r)
	if current.Role == "superadmin" {
		return true
	}
	if s.users != nil {
		if user, ok := s.users.FindByEmail(current.Email); ok {
			if user.OrganizationRoles[organizationID] == "organization_admin" {
				return true
			}
		}
	}
	return current.Role == "organization_admin" && containsString(current.OrganizationIDs, organizationID)
}

func (s *Server) projectRole(r *http.Request, projectID string) string {
	current := s.sessionFromRequest(r)
	if current.Role == "superadmin" {
		return "superadmin"
	}
	if s.users != nil {
		if user, ok := s.users.FindByEmail(current.Email); ok {
			if user.OrganizationRoles[repo.OrganizationIDFrom(r.Context())] == "organization_admin" || user.Role == "organization_admin" {
				return "organization_admin"
			}
			return user.ProjectRoles[projectID]
		}
	}
	if current.Role == "organization_admin" {
		return "organization_admin"
	}
	return ""
}

func (s *Server) canAccessProject(r *http.Request, projectID string) bool {
	role := s.projectRole(r, projectID)
	return role == "superadmin" || role == "organization_admin" || role == "project_admin" || role == "project_member"
}

func (s *Server) canManageProject(r *http.Request, projectID string) bool {
	role := s.projectRole(r, projectID)
	return role == "superadmin" || role == "organization_admin" || role == "project_admin"
}

func (s *Server) requireProjectPathAccess(next http.Handler, manage bool) http.Handler {
	return s.requireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		projectID := r.PathValue("id")
		if _, err := s.projects.Get(r.Context(), projectID); err != nil {
			s.writeServiceError(w, err)
			return
		}
		allowed := s.canAccessProject(r, projectID)
		if manage {
			allowed = s.canManageProject(r, projectID)
		}
		if !allowed {
			writeError(w, http.StatusForbidden, errors.New("没有访问该项目的权限"))
			return
		}
		ctx := repo.WithProject(r.Context(), projectID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}))
}

func (s *Server) requireCurrentProjectAccess(next http.Handler, manage bool) http.Handler {
	return s.requireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		projectID := s.sessions.projectID(r)
		if projectID == "" {
			writeError(w, http.StatusBadRequest, errors.New("请先选择项目"))
			return
		}
		if _, err := s.projects.Get(r.Context(), projectID); err != nil {
			s.writeServiceError(w, err)
			return
		}
		allowed := s.canAccessProject(r, projectID)
		if manage {
			allowed = s.canManageProject(r, projectID)
		}
		if !allowed {
			writeError(w, http.StatusForbidden, errors.New("没有访问该项目的权限"))
			return
		}
		ctx := repo.WithProject(r.Context(), projectID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}))
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

func (s *Server) authenticatedSession(r *http.Request) bool {
	if !s.validSession(r) {
		return false
	}
	current := s.sessionFromRequest(r)
	if s.users == nil {
		return true
	}
	_, ok := s.users.FindByEmail(current.Email)
	return ok
}

func (s *sessionStore) organizationID(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return repo.DefaultOrganizationID
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.sessions[cookie.Value]
	if !ok || time.Now().After(current.ExpiresAt) {
		return repo.DefaultOrganizationID
	}
	if current.OrganizationID == "" {
		return repo.DefaultOrganizationID
	}
	return current.OrganizationID
}

func (s *sessionStore) setOrganization(r *http.Request, organizationID string) bool {
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
	current.OrganizationID = organizationID
	current.ProjectID = ""
	s.sessions[cookie.Value] = current
	if s.persist() != nil {
		return false
	}
	return true
}

func (s *sessionStore) projectID(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.sessions[cookie.Value]
	if !ok || time.Now().After(current.ExpiresAt) {
		return ""
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
		if !s.authenticatedSession(r) {
			writeError(w, http.StatusUnauthorized, errors.New("authentication required"))
			return
		}
		organizationID := s.sessions.organizationID(r)
		if requestedOrganizationID := strings.TrimSpace(r.Header.Get("X-SnakeRule-Organization-ID")); requestedOrganizationID != "" {
			if !s.canAccessOrganization(r, requestedOrganizationID) {
				writeError(w, http.StatusForbidden, errors.New("没有访问该组织的权限"))
				return
			}
			organizationID = requestedOrganizationID
		}
		ctx := repo.WithProject(repo.WithOrganization(r.Context(), organizationID), s.sessions.projectID(r))
		ctx = context.WithValue(ctx, invocationCredentialContextKey{}, invocationCredential{Type: "session"})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) requireOrganizationSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		organizationID := r.PathValue("organizationID")
		if organizationID == "" {
			writeError(w, http.StatusBadRequest, errors.New("organization id is required"))
			return
		}
		r.Header.Set("X-SnakeRule-Organization-ID", organizationID)
		s.requireSession(next).ServeHTTP(w, r)
	})
}

func (s *Server) requireSuperAdmin(next http.Handler) http.Handler {
	return s.requireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.isSuperAdmin(r) {
			writeError(w, http.StatusForbidden, errors.New("仅超级管理员可访问"))
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func (s *Server) requireSessionOrProjectKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.authenticatedSession(r) {
			ctx := repo.WithProject(repo.WithOrganization(r.Context(), s.sessions.organizationID(r)), s.sessions.projectID(r))
			projectID := s.sessions.projectID(r)
			if projectID != "" && !s.canAccessProject(r.WithContext(ctx), projectID) {
				writeError(w, http.StatusForbidden, errors.New("没有访问该项目的权限"))
				return
			}
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
		ctx := repo.WithProject(repo.WithOrganization(r.Context(), credential.Organization), credential.Project)
		ctx = context.WithValue(ctx, projectKeyAuthContextKey{}, true)
		ctx = context.WithValue(ctx, invocationCredentialContextKey{}, invocationCredential{ID: credential.ID, Type: credential.Type})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
