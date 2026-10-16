package evaluator

import (
	"encoding/json"
	"testing"

	"github.com/szplanner/snakerule/internal/domain"
)

func TestEvaluateNestedConditions(t *testing.T) {
	facts := map[string]any{
		"order":    map[string]any{"amount": json.Number("12000")},
		"customer": map[string]any{"isNew": true, "tier": "standard"},
	}
	condition := domain.Condition{All: []domain.Condition{
		{Fact: "order.amount", Operator: "gte", Value: 10000},
		{Any: []domain.Condition{
			{Fact: "customer.tier", Operator: "equal", Value: "vip"},
			{Fact: "customer.isNew", Operator: "eq", Value: true},
		}},
		{Not: &domain.Condition{Fact: "customer.tier", Operator: "equal", Value: "blocked"}},
	}}

	matched, err := Evaluate(condition, facts)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !matched {
		t.Fatal("Evaluate() = false, want true")
	}
}

func TestEvaluateMissingFactsAndNullOperators(t *testing.T) {
	facts := map[string]any{"value": nil}
	tests := []struct {
		name      string
		condition domain.Condition
		want      bool
	}{
		{name: "missing is null", condition: domain.Condition{Fact: "missing", Operator: "isNull"}, want: true},
		{name: "nil is null", condition: domain.Condition{Fact: "value", Operator: "isNull"}, want: true},
		{name: "missing is not null", condition: domain.Condition{Fact: "missing", Operator: "isNotNull"}, want: false},
		{name: "missing is not equal", condition: domain.Condition{Fact: "missing", Operator: "notEqual", Value: 1}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Evaluate(tt.condition, facts)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluateMembershipAndContains(t *testing.T) {
	facts := map[string]any{
		"labels": []any{"new", "review"},
		"name":   "order-risk",
		"state":  "review",
	}
	tests := []struct {
		name      string
		condition domain.Condition
		want      bool
	}{
		{name: "in", condition: domain.Condition{Fact: "state", Operator: "in", Value: []any{"new", "review"}}, want: true},
		{name: "not in", condition: domain.Condition{Fact: "state", Operator: "notIn", Value: []any{"new", "vip"}}, want: true},
		{name: "array contains", condition: domain.Condition{Fact: "labels", Operator: "contains", Value: "review"}, want: true},
		{name: "string contains", condition: domain.Condition{Fact: "name", Operator: "contains", Value: "risk"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Evaluate(tt.condition, facts)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluateRejectsInvalidConditions(t *testing.T) {
	tests := []struct {
		name      string
		condition domain.Condition
		facts     map[string]any
	}{
		{name: "no branch", condition: domain.Condition{}},
		{name: "multiple branches", condition: domain.Condition{All: []domain.Condition{}, Fact: "value", Operator: "equal"}},
		{name: "unsupported operator", condition: domain.Condition{Fact: "value", Operator: "matches"}},
		{name: "in expects array", condition: domain.Condition{Fact: "value", Operator: "in", Value: "not-an-array"}, facts: map[string]any{"value": "x"}},
		{name: "numeric comparison rejects strings", condition: domain.Condition{Fact: "value", Operator: "greaterThan", Value: 1}, facts: map[string]any{"value": "many"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Evaluate(tt.condition, tt.facts); err == nil {
				t.Fatal("Evaluate() error = nil, want an error")
			}
		})
	}
}
