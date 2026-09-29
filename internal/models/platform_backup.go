package models

import "time"

// UserBackup includes password hash for disaster-recovery restore.
type UserBackup struct {
	ID           string   `json:"id"`
	Username     string   `json:"username"`
	Name         string   `json:"name"`
	Email        string   `json:"email"`
	PasswordHash string   `json:"password_hash"`
	MFAEnabled   bool     `json:"mfa_enabled"`
	Role         UserRole `json:"role"`
	TenantID     string   `json:"tenant_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func UserBackupFromUser(u User) UserBackup {
	return UserBackup{
		ID:           u.ID,
		Username:     u.Username,
		Name:         u.Name,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		MFAEnabled:   u.MFAEnabled,
		Role:         u.Role,
		TenantID:     u.TenantID,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

func (b UserBackup) ToUser() User {
	return User{
		ID:           b.ID,
		Username:     b.Username,
		Name:         b.Name,
		Email:        b.Email,
		PasswordHash: b.PasswordHash,
		MFAEnabled:   b.MFAEnabled,
		Role:         b.Role,
		TenantID:     b.TenantID,
		CreatedAt:    b.CreatedAt,
		UpdatedAt:    b.UpdatedAt,
	}
}

// HostBackup includes ingest credentials for agent continuity.
type HostBackup struct {
	Host
	IngestToken string `json:"ingest_token,omitempty"`
	TokenHash   string `json:"token_hash,omitempty"`
}

// SlackSettingBackup is one tenant's Slack integration (empty tenant = platform).
type SlackSettingBackup struct {
	TenantID string      `json:"tenant_id"`
	Config   SlackConfig `json:"config"`
}

// PlatformSettingsBackup is optional platform/tenant settings bundle.
type PlatformSettingsBackup struct {
	Org        *OrgSettings         `json:"org,omitempty"`
	SMTP       *SMTPConfig          `json:"smtp,omitempty"`
	Webhooks   []WebhookConfig      `json:"webhooks,omitempty"`
	Server     *ServerSettings      `json:"server,omitempty"`
	StatusPage *StatusPageConfig    `json:"status_page,omitempty"`
	Slack      []SlackSettingBackup `json:"slack,omitempty"`
}
