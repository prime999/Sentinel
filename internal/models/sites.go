package models

import "time"

type Site struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	TenantID     string    `json:"tenant_id,omitempty"`
	PrimaryHost  string    `json:"primary_host"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
