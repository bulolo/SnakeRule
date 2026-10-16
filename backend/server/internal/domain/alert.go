package domain

import "time"

// AlertChannel is a platform-wide destination that projects may reference.
// Secret is never exposed by the API.
type AlertChannel struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	URL       string    `json:"url"`
	Enabled   bool      `json:"enabled"`
	HasSecret bool      `json:"has_secret"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Secret    string    `json:"-"`
}

type APIKeyCredential struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Prefix    string    `json:"prefix"`
	Hash      string    `json:"-"`
	Workspace string    `json:"-"`
	Project   string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}
