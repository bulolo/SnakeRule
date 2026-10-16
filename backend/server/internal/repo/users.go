package repo

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/szplanner/snakerule/internal/domain"
)

type PersistentUserRepository struct {
	mu    sync.RWMutex
	path  string
	items map[string]domain.User
}
type persistedUser struct {
	domain.User
	Hash string `json:"password_hash"`
	Salt string `json:"password_salt"`
}

func NewPersistentUserRepository(path string) (*PersistentUserRepository, error) {
	r := &PersistentUserRepository{path: path, items: map[string]domain.User{}}
	var stored map[string]persistedUser
	if err := loadPersistent(path, &stored); err != nil {
		return nil, err
	}
	for id, p := range stored {
		u := p.User
		u.PasswordHash = p.Hash
		u.PasswordSalt = p.Salt
		r.items[id] = u
	}
	return r, nil
}
func hashPassword(password, salt string) string {
	key := []byte(password)
	block := []byte(salt)
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(append(block, 0, 0, 0, 1))
	u := mac.Sum(nil)
	out := append([]byte(nil), u...)
	for i := 1; i < 120000; i++ {
		mac = hmac.New(sha256.New, key)
		_, _ = mac.Write(u)
		u = mac.Sum(nil)
		for j := range out {
			out[j] ^= u[j]
		}
	}
	return base64.RawStdEncoding.EncodeToString(out)
}
func (r *PersistentUserRepository) persist() error {
	stored := make(map[string]persistedUser, len(r.items))
	for id, u := range r.items {
		stored[id] = persistedUser{User: u, Hash: u.PasswordHash, Salt: u.PasswordSalt}
	}
	return savePersistent(r.path, stored)
}
func (r *PersistentUserRepository) SeedSuperAdmin(email, name, password string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return errors.New("superadmin email and password are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.items {
		if strings.EqualFold(u.Email, email) {
			return nil
		}
	}
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return err
	}
	salt := base64.RawStdEncoding.EncodeToString(saltBytes)
	idBytes := make([]byte, 12)
	if _, err := rand.Read(idBytes); err != nil {
		return err
	}
	id := base64.RawURLEncoding.EncodeToString(idBytes)
	r.items[id] = domain.User{ID: id, Email: email, Name: name, Role: "superadmin", OrganizationIDs: []string{"default"}, CreatedAt: time.Now().UTC(), PasswordSalt: salt, PasswordHash: hashPassword(password, salt)}
	return r.persist()
}
func (r *PersistentUserRepository) List() []domain.User {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.User, 0, len(r.items))
	for _, u := range r.items {
		u.PasswordHash = ""
		u.PasswordSalt = ""
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Email < out[j].Email })
	return out
}
func (r *PersistentUserRepository) FindByEmail(email string) (domain.User, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.items {
		if strings.EqualFold(u.Email, strings.TrimSpace(email)) {
			return u, true
		}
	}
	return domain.User{}, false
}
func (r *PersistentUserRepository) FindByID(id string) (domain.User, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.items[id]
	return u, ok
}
func (r *PersistentUserRepository) Verify(email, password string) (domain.User, bool) {
	u, ok := r.FindByEmail(email)
	if !ok {
		return domain.User{}, false
	}
	candidate := hashPassword(password, u.PasswordSalt)
	return u, subtle.ConstantTimeCompare([]byte(candidate), []byte(u.PasswordHash)) == 1
}
func (r *PersistentUserRepository) Save(u domain.User, password string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if strings.TrimSpace(u.Email) == "" || strings.TrimSpace(u.Name) == "" {
		return errors.New("name and email are required")
	}
	for id, current := range r.items {
		if id != u.ID && strings.EqualFold(current.Email, u.Email) {
			return errors.New("email already exists")
		}
	}
	if u.ID == "" {
		b := make([]byte, 12)
		if _, err := rand.Read(b); err != nil {
			return err
		}
		u.ID = base64.RawURLEncoding.EncodeToString(b)
		u.CreatedAt = time.Now().UTC()
	}
	if password != "" {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			return err
		}
		u.PasswordSalt = base64.RawStdEncoding.EncodeToString(b)
		u.PasswordHash = hashPassword(password, u.PasswordSalt)
	} else if old, ok := r.items[u.ID]; ok {
		u.PasswordHash = old.PasswordHash
		u.PasswordSalt = old.PasswordSalt
	} else {
		return errors.New("password is required")
	}
	if u.Role != "organization_admin" && u.Role != "organization_member" && u.Role != "superadmin" {
		return errors.New("invalid user role")
	}
	if previous, ok := r.items[u.ID]; ok && previous.Role == "superadmin" && u.Role != "superadmin" {
		count := 0
		for _, current := range r.items {
			if current.Role == "superadmin" {
				count++
			}
		}
		if count <= 1 {
			return errors.New("至少保留一个超级管理员")
		}
	}
	r.items[u.ID] = u
	return r.persist()
}
func (r *PersistentUserRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.items[id]
	if !ok {
		return ErrNotFound
	}
	if u.Role == "superadmin" {
		return ErrProtected
	}
	delete(r.items, id)
	return r.persist()
}
