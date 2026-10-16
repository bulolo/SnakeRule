package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/szplanner/snakerule/internal/domain"
	"github.com/szplanner/snakerule/internal/engine/evaluator"
	"github.com/szplanner/snakerule/internal/repo"
)

var ErrInvalid = errors.New("invalid request")

type RulesetService struct{ repository repo.RulesetRepository }

func NewRulesetService(repository repo.RulesetRepository) *RulesetService {
	return &RulesetService{repository: repository}
}
func (s *RulesetService) List(ctx context.Context) ([]domain.Ruleset, error) {
	return s.repository.List(ctx)
}
func (s *RulesetService) Get(ctx context.Context, id string) (domain.Ruleset, error) {
	return s.repository.Get(ctx, id)
}
func (s *RulesetService) Delete(ctx context.Context, id string) error {
	return s.repository.Delete(ctx, id)
}

func (s *RulesetService) ListVersions(ctx context.Context, id string) ([]domain.RulesetVersion, error) {
	return s.repository.ListVersions(ctx, id)
}

func (s *RulesetService) Publish(ctx context.Context, id, actor, note string) (domain.RulesetVersion, error) {
	item, err := s.repository.Get(ctx, id)
	if err != nil {
		return domain.RulesetVersion{}, err
	}
	if _, err := normalizeRuleset(item); err != nil {
		return domain.RulesetVersion{}, err
	}
	return s.repository.Publish(ctx, id, actor, note)
}

func (s *RulesetService) Rollback(ctx context.Context, id string, version int, actor string) (domain.RulesetVersion, error) {
	item, err := s.repository.GetVersion(ctx, id, version)
	if err != nil {
		return domain.RulesetVersion{}, err
	}
	restored := item.Snapshot
	if _, err := normalizeRuleset(restored); err != nil {
		return domain.RulesetVersion{}, err
	}
	return s.repository.Rollback(ctx, id, version, actor)
}
func (s *RulesetService) Save(ctx context.Context, item domain.Ruleset) error {
	item, err := normalizeRuleset(item)
	if err != nil {
		return err
	}
	return s.repository.Save(ctx, item)
}

func normalizeRuleset(item domain.Ruleset) (domain.Ruleset, error) {
	if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.Name) == "" {
		return domain.Ruleset{}, fmt.Errorf("%w: ruleset id and name are required", ErrInvalid)
	}
	if item.Version == "" {
		item.Version = "1.0.0"
	}
	ids := make(map[string]struct{}, len(item.Rules))
	for i := range item.Rules {
		rule := &item.Rules[i]
		if strings.TrimSpace(rule.ID) == "" || strings.TrimSpace(rule.Name) == "" {
			return domain.Ruleset{}, fmt.Errorf("%w: rule id and name are required", ErrInvalid)
		}
		if _, exists := ids[rule.ID]; exists {
			return domain.Ruleset{}, fmt.Errorf("%w: duplicate rule id %q", ErrInvalid, rule.ID)
		}
		ids[rule.ID] = struct{}{}
		if err := evaluator.Validate(rule.Conditions); err != nil {
			return domain.Ruleset{}, fmt.Errorf("%w: rule %q: %v", ErrInvalid, rule.ID, err)
		}
	}
	sort.SliceStable(item.Rules, func(i, j int) bool { return item.Rules[i].Priority > item.Rules[j].Priority })
	return item, nil
}

type Engine struct{ rulesets repo.RulesetRepository }

func NewEngine(rulesets repo.RulesetRepository) *Engine { return &Engine{rulesets: rulesets} }
func (s *Engine) Evaluate(ctx context.Context, id string, facts map[string]any) (domain.Decision, error) {
	set, err := s.rulesets.GetPublished(ctx, id)
	if err != nil {
		return domain.Decision{}, err
	}
	return evaluateRuleset(set, facts)
}

func (s *Engine) EvaluateDraft(set domain.Ruleset, facts map[string]any) (domain.Decision, error) {
	set, err := normalizeRuleset(set)
	if err != nil {
		return domain.Decision{}, err
	}
	set.PublishedVersion = 0
	return evaluateRuleset(set, facts)
}

func evaluateRuleset(set domain.Ruleset, facts map[string]any) (domain.Decision, error) {
	decision := domain.Decision{RulesetID: set.ID, Version: set.Version, PublishedVersion: set.PublishedVersion, HitRules: []domain.MatchedRule{}}
	if !set.Enabled {
		return decision, nil
	}
	for _, rule := range set.Rules {
		if !rule.Enabled {
			continue
		}
		matched, err := evaluator.Evaluate(rule.Conditions, facts)
		if err != nil {
			return domain.Decision{}, fmt.Errorf("evaluate rule %q: %w", rule.ID, err)
		}
		if matched {
			decision.Matched = true
			decision.HitRules = append(decision.HitRules, domain.MatchedRule{RuleID: rule.ID, Name: rule.Name, Priority: rule.Priority, Actions: rule.Actions})
		}
	}
	return decision, nil
}
