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

type Rulesets struct{ repository repo.RulesetRepository }

func NewRulesets(repository repo.RulesetRepository) *Rulesets {
	return &Rulesets{repository: repository}
}
func (s *Rulesets) List(ctx context.Context) ([]domain.Ruleset, error) { return s.repository.List(ctx) }
func (s *Rulesets) Get(ctx context.Context, id string) (domain.Ruleset, error) {
	return s.repository.Get(ctx, id)
}
func (s *Rulesets) Delete(ctx context.Context, id string) error { return s.repository.Delete(ctx, id) }
func (s *Rulesets) Save(ctx context.Context, item domain.Ruleset) error {
	if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.Name) == "" {
		return fmt.Errorf("%w: ruleset id and name are required", ErrInvalid)
	}
	if item.Version == "" {
		item.Version = "1.0.0"
	}
	ids := make(map[string]struct{}, len(item.Rules))
	for i := range item.Rules {
		rule := &item.Rules[i]
		if strings.TrimSpace(rule.ID) == "" || strings.TrimSpace(rule.Name) == "" {
			return fmt.Errorf("%w: rule id and name are required", ErrInvalid)
		}
		if _, exists := ids[rule.ID]; exists {
			return fmt.Errorf("%w: duplicate rule id %q", ErrInvalid, rule.ID)
		}
		ids[rule.ID] = struct{}{}
		if err := evaluator.Validate(rule.Conditions); err != nil {
			return fmt.Errorf("%w: rule %q: %v", ErrInvalid, rule.ID, err)
		}
	}
	sort.SliceStable(item.Rules, func(i, j int) bool { return item.Rules[i].Priority > item.Rules[j].Priority })
	return s.repository.Save(ctx, item)
}

type Engine struct{ rulesets repo.RulesetRepository }

func NewEngine(rulesets repo.RulesetRepository) *Engine { return &Engine{rulesets: rulesets} }
func (s *Engine) Evaluate(ctx context.Context, id string, facts map[string]any) (domain.Decision, error) {
	set, err := s.rulesets.Get(ctx, id)
	if err != nil {
		return domain.Decision{}, err
	}
	decision := domain.Decision{RulesetID: set.ID, Version: set.Version, HitRules: []domain.MatchedRule{}}
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
