package domain

import "time"

type Notification struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Project   string    `json:"project"`
	Ruleset   string    `json:"ruleset"`
	CreatedAt time.Time `json:"created_at"`
	Read      bool      `json:"read"`
}
