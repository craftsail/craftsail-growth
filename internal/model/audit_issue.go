// SPDX-License-Identifier: AGPL-3.0-or-later

package model

// AuditIssue is one finding from one audit run. Code refers to the issue
// registry in internal/service/audit/issues.go.
type AuditIssue struct {
	ID        uint64         `gorm:"primaryKey" json:"id"`
	ProjectID uint64         `gorm:"index:idx_issue_code,priority:1;not null" json:"project_id"`
	AuditID   uint64         `gorm:"index:idx_issue_audit,priority:1;not null" json:"audit_id"`
	PageID    *uint64        `json:"page_id"`
	URL       string         `gorm:"size:1024" json:"url"`
	Code      string         `gorm:"size:48;index:idx_issue_code,priority:2;not null" json:"code"`
	Severity  string         `gorm:"size:12;index:idx_issue_audit,priority:2;not null" json:"severity"`
	Layer     string         `gorm:"size:12;not null" json:"layer"`
	Blocked   bool           `gorm:"not null;default:false" json:"blocked"`
	Detail    map[string]any `gorm:"serializer:json" json:"detail"`
	CreatedAt int64          `json:"created_at"`
}
