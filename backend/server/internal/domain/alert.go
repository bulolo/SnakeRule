package domain

import "time"

// AlertChannel is a organization-scoped destination that projects may reference.
// Secret is never exposed by the API.
type AlertChannel struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id,omitempty"`
	Name           string    `json:"name"`
	Type           string    `json:"type"`
	URL            string    `json:"url"`
	Enabled        bool      `json:"enabled"`
	HasSecret      bool      `json:"has_secret"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Secret         string    `json:"-"`
}

type APIKeyCredential struct {
	ID                     string     `json:"id"`
	Name                   string     `json:"name"`
	Type                   string     `json:"type"`
	Prefix                 string     `json:"prefix"`
	Hash                   string     `json:"-"`
	CredentialKind         string     `json:"credential_kind,omitempty"`
	ExpiresAt              *time.Time `json:"expires_at,omitempty"`
	SigningSecretEncrypted string     `json:"-"`
	Organization           string     `json:"-"`
	Project                string     `json:"-"`
	CreatedAt              time.Time  `json:"created_at"`
}
