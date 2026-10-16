package bootstrap

import (
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
)

func loadOrCreateJWTSigningKey(path string) ([]byte, error) {
	if key, err := os.ReadFile(path); err == nil {
		if len(key) < 32 {
			return nil, errors.New("JWT signing key must be at least 32 bytes")
		}
		return key, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, key, 0600); err != nil {
		return nil, err
	}
	return key, nil
}
