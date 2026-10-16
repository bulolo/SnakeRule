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

type OrganizationService struct {
	mu         sync.Mutex
	repository repo.OrganizationRepository
}

var organizationSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

func NewOrganizationService(repository repo.OrganizationRepository) *OrganizationService {
	return &OrganizationService{repository: repository}
}

func (s *OrganizationService) List(ctx context.Context) ([]domain.Organization, error) {
	return s.repository.List(ctx)
}

func (s *OrganizationService) Get(ctx context.Context, id string) (domain.Organization, error) {
	return s.repository.Get(ctx, id)
}

func (s *OrganizationService) Create(ctx context.Context, name, slug, description string, adminUserID string) (domain.Organization, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 80 {
		return domain.Organization{}, fmt.Errorf("%w: organization name must contain 1 to 80 characters", ErrInvalid)
	}
	idBytes := make([]byte, 8)
	if _, err := rand.Read(idBytes); err != nil {
		return domain.Organization{}, err
	}
	slug = strings.TrimSpace(strings.ToLower(slug))
	if slug == "" {
		slug = "org-" + hex.EncodeToString(idBytes[:4])
	}
	if !organizationSlugPattern.MatchString(slug) {
		return domain.Organization{}, fmt.Errorf("%w: slug must use 1 to 32 lowercase letters, digits, hyphens or underscores", ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureSlugAvailable(ctx, slug, ""); err != nil {
		return domain.Organization{}, err
	}
	now := time.Now().UTC()
	item := domain.Organization{ID: "org-" + hex.EncodeToString(idBytes), Name: name, Slug: slug, Description: strings.TrimSpace(description), CreatedAt: now, UpdatedAt: now}
	item.AdminUserID = adminUserID
	if err := s.repository.Save(ctx, item); err != nil {
		return domain.Organization{}, err
	}
	return item, nil
}

func (s *OrganizationService) Update(ctx context.Context, id, name, slug, description string) (domain.Organization, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 80 {
		return domain.Organization{}, fmt.Errorf("%w: organization name must contain 1 to 80 characters", ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, err := s.repository.Get(ctx, id)
	if err != nil {
		return domain.Organization{}, err
	}
	slug = strings.TrimSpace(strings.ToLower(slug))
	if id == repo.DefaultOrganizationID {
		slug = "default"
	}
	if !organizationSlugPattern.MatchString(slug) {
		return domain.Organization{}, fmt.Errorf("%w: slug must use 1 to 32 lowercase letters, digits, hyphens or underscores", ErrInvalid)
	}
	if err := s.ensureSlugAvailable(ctx, slug, id); err != nil {
		return domain.Organization{}, err
	}
	item.Name = name
	item.Slug = slug
	item.Description = strings.TrimSpace(description)
	item.UpdatedAt = time.Now().UTC()
	if err := s.repository.Save(ctx, item); err != nil {
		return domain.Organization{}, err
	}
	return item, nil
}

func (s *OrganizationService) ensureSlugAvailable(ctx context.Context, slug, exceptID string) error {
	items, err := s.repository.List(ctx)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.ID != exceptID && item.Slug == slug {
			return fmt.Errorf("%w: organization slug %q is already in use", ErrInvalid, slug)
		}
	}
	return nil
}

func (s *OrganizationService) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.repository.Delete(ctx, id)
}
