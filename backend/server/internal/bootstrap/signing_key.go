package bootstrap

import (
	"crypto/rand"
	"errors"

	"github.com/szplanner/snakerule/internal/repo"
)

func loadOrCreateJWTSigningKey(path string) ([]byte, error) {
	var stored struct {
		Key []byte `json:"key"`
	}
	if err := repo.LoadPersistent(path, &stored); err != nil {
		return nil, err
	}
	if len(stored.Key) > 0 {
		if len(stored.Key) < 32 {
			return nil, errors.New("JWT signing key must be at least 32 bytes")
		}
		return stored.Key, nil
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := repo.SavePersistent(path, struct {
		Key []byte `json:"key"`
	}{Key: key}); err != nil {
		return nil, err
	}
	return key, nil
}
