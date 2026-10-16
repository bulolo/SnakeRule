package domain

import "time"

type User struct {
	ID                string            `json:"id"`
	Username          string            `json:"username,omitempty"`
	Email             string            `json:"email"`
	Name              string            `json:"name"`
	Role              string            `json:"role"`
	OrganizationIDs   []string          `json:"organization_ids"`
	OrganizationRoles map[string]string `json:"organization_roles,omitempty"`
	ProjectRoles      map[string]string `json:"project_roles,omitempty"`
	CreatedAt         time.Time         `json:"created_at"`
	PasswordHash      string            `json:"-"`
	PasswordSalt      string            `json:"-"`
}
