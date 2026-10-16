package repo

import (
	"context"
	"strings"
	"sync"
)

type PlatformSettings struct {
	Name    string `json:"name"`
	Tagline string `json:"tagline"`
	LogoURL string `json:"logo_url"`
}

type PlatformSettingsRepository interface {
	Get(context.Context) (PlatformSettings, error)
	Save(context.Context, PlatformSettings) error
}

type PersistentPlatformSettingsRepository struct {
	mu    sync.RWMutex
	path  string
	value PlatformSettings
}

func NewPersistentPlatformSettingsRepository(path string) (*PersistentPlatformSettingsRepository, error) {
	r := &PersistentPlatformSettingsRepository{path: path, value: PlatformSettings{Name: "SnakeRule", Tagline: "确定性规则求值平台"}}
	if err := loadPersistent(path, &r.value); err != nil {
		return nil, err
	}
	if strings.TrimSpace(r.value.Name) == "" {
		r.value.Name = "SnakeRule"
	}
	if strings.TrimSpace(r.value.Tagline) == "" {
		r.value.Tagline = "确定性规则求值平台"
	}
	if err := savePersistent(path, r.value); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *PersistentPlatformSettingsRepository) Get(context.Context) (PlatformSettings, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.value, nil
}

func (r *PersistentPlatformSettingsRepository) Save(_ context.Context, value PlatformSettings) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := savePersistent(r.path, value); err != nil {
		return err
	}
	r.value = value
	return nil
}
