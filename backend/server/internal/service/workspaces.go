package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/szplanner/snakerule/internal/domain"
	"github.com/szplanner/snakerule/internal/repo"
)

type Workspaces struct {
	mu         sync.Mutex
	repository repo.WorkspaceRepository
}

var workspaceSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

func NewWorkspaces(repository repo.WorkspaceRepository) *Workspaces {
	return &Workspaces{repository: repository}
}

func (s *Workspaces) List(ctx context.Context) ([]domain.Workspace, error) {
	return s.repository.List(ctx)
}

func (s *Workspaces) Get(ctx context.Context, id string) (domain.Workspace, error) {
	return s.repository.Get(ctx, id)
}

func (s *Workspaces) Create(ctx context.Context, name, slug, description string) (domain.Workspace, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 80 {
		return domain.Workspace{}, fmt.Errorf("%w: workspace name must contain 1 to 80 characters", ErrInvalid)
	}
	idBytes := make([]byte, 8)
	if _, err := rand.Read(idBytes); err != nil {
		return domain.Workspace{}, err
	}
	slug = strings.TrimSpace(strings.ToLower(slug))
	if slug == "" {
		slug = "ws-" + hex.EncodeToString(idBytes[:4])
	}
	if !workspaceSlugPattern.MatchString(slug) {
		return domain.Workspace{}, fmt.Errorf("%w: slug must use 1 to 32 lowercase letters, digits, hyphens or underscores", ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureSlugAvailable(ctx, slug, ""); err != nil {
		return domain.Workspace{}, err
	}
	now := time.Now().UTC()
	item := domain.Workspace{ID: "ws-" + hex.EncodeToString(idBytes), Name: name, Slug: slug, Description: strings.TrimSpace(description), CreatedAt: now, UpdatedAt: now}
	if err := s.repository.Save(ctx, item); err != nil {
		return domain.Workspace{}, err
	}
	return item, nil
}

func (s *Workspaces) Update(ctx context.Context, id, name, slug, description string) (domain.Workspace, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 80 {
		return domain.Workspace{}, fmt.Errorf("%w: workspace name must contain 1 to 80 characters", ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, err := s.repository.Get(ctx, id)
	if err != nil {
		return domain.Workspace{}, err
	}
	slug = strings.TrimSpace(strings.ToLower(slug))
	if id == repo.DefaultWorkspaceID {
		slug = "default"
	}
	if !workspaceSlugPattern.MatchString(slug) {
		return domain.Workspace{}, fmt.Errorf("%w: slug must use 1 to 32 lowercase letters, digits, hyphens or underscores", ErrInvalid)
	}
	if err := s.ensureSlugAvailable(ctx, slug, id); err != nil {
		return domain.Workspace{}, err
	}
	item.Name = name
	item.Slug = slug
	item.Description = strings.TrimSpace(description)
	item.UpdatedAt = time.Now().UTC()
	if err := s.repository.Save(ctx, item); err != nil {
		return domain.Workspace{}, err
	}
	return item, nil
}

func (s *Workspaces) ensureSlugAvailable(ctx context.Context, slug, exceptID string) error {
	items, err := s.repository.List(ctx)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.ID != exceptID && item.Slug == slug {
			return fmt.Errorf("%w: workspace slug %q is already in use", ErrInvalid, slug)
		}
	}
	return nil
}

func (s *Workspaces) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.repository.Delete(ctx, id)
}
