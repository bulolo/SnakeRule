package domain

import "time"

// Workspace groups rulesets for an independent team or project area.
type Workspace struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Ruleset is a versioned collection of rules evaluated as one decision.
type Ruleset struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
	Rules       []Rule `json:"rules"`
}

type Rule struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Priority    int       `json:"priority"`
	Enabled     bool      `json:"enabled"`
	Conditions  Condition `json:"conditions"`
	Actions     []Action  `json:"actions,omitempty"`
}

// Condition supports a recursive JSON DSL: all, any, not, or a fact predicate.
type Condition struct {
	All      []Condition `json:"all,omitempty"`
	Any      []Condition `json:"any,omitempty"`
	Not      *Condition  `json:"not,omitempty"`
	Fact     string      `json:"fact,omitempty"`
	Operator string      `json:"operator,omitempty"`
	Value    any         `json:"value,omitempty"`
}

// Action is declarative for now. The executor layer will add controlled side effects later.
type Action struct {
	Type   string         `json:"type"`
	Config map[string]any `json:"config,omitempty"`
}

type Decision struct {
	RulesetID string        `json:"rulesetId"`
	Version   string        `json:"version"`
	Matched   bool          `json:"matched"`
	HitRules  []MatchedRule `json:"hitRules"`
}

type MatchedRule struct {
	RuleID   string   `json:"ruleId"`
	Name     string   `json:"name"`
	Priority int      `json:"priority"`
	Actions  []Action `json:"actions,omitempty"`
}
