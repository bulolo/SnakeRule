package domain

import "time"

// InvocationLog records rule evaluations and bounded, redacted request/response previews.
type InvocationLog struct {
	ID             string              `json:"id"`
	OrganizationID string              `json:"organization_id"`
	ProjectID      string              `json:"project_id"`
	RulesetID      string              `json:"ruleset_id,omitempty"`
	Method         string              `json:"method"`
	Path           string              `json:"path"`
	Source         string              `json:"source"`
	CredentialType string              `json:"credential_type"`
	StatusCode     int                 `json:"status_code"`
	Matched        bool                `json:"matched"`
	HitRules       []InvocationHitRule `json:"hit_rules,omitempty"`
	RequestBody    string              `json:"request_body,omitempty"`
	ResponseBody   string              `json:"response_body,omitempty"`
	DurationMS     int64               `json:"duration_ms"`
	CreatedAt      time.Time           `json:"created_at"`
}

type InvocationHitRule struct {
	RuleID   string   `json:"rule_id"`
	Name     string   `json:"name"`
	Priority int      `json:"priority"`
	Actions  []string `json:"actions,omitempty"`
}
