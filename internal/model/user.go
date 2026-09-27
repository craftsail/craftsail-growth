// SPDX-License-Identifier: AGPL-3.0-or-later

package model

// Workspace roles. Admins can do everything; members only reach the
// projects shared with them.
const (
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// Project access levels for members. "" means the project is not shared.
const (
	AccessView = "view"
	AccessEdit = "edit"
)

type User struct {
	ID           uint64 `gorm:"primaryKey" json:"id"`
	Username     string `gorm:"size:64;not null;uniqueIndex:idx_users_username" json:"username"`
	PasswordHash string `gorm:"size:100;not null" json:"-"`
	Role         string `gorm:"size:16;not null;default:member" json:"role"`
	Disabled     bool   `gorm:"not null;default:false" json:"disabled"`
	// MustChangePassword is set on the built-in default account until its
	// password is changed; the dashboard shows a reminder meanwhile.
	MustChangePassword bool   `gorm:"not null;default:false" json:"must_change_password"`
	LastLoginAt        *int64 `json:"last_login_at"`
	CreatedAt          int64  `json:"created_at"`
	UpdatedAt          int64  `json:"updated_at"`
}

// ProjectMember shares one project with one member.
type ProjectMember struct {
	ID        uint64 `gorm:"primaryKey" json:"id"`
	UserID    uint64 `gorm:"not null;uniqueIndex:idx_project_members_pair" json:"user_id"`
	ProjectID uint64 `gorm:"not null;uniqueIndex:idx_project_members_pair;index" json:"project_id"`
	Access    string `gorm:"size:8;not null" json:"access"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// Session is a signed-in browser. Only the SHA-256 of the cookie value is
// stored, so a database leak does not hand out live sessions.
type Session struct {
	ID        uint64 `gorm:"primaryKey"`
	TokenHash string `gorm:"size:64;not null;uniqueIndex:idx_sessions_token"`
	UserID    uint64 `gorm:"not null;index"`
	ExpiresAt int64  `gorm:"not null;index"`
	CreatedAt int64
}
