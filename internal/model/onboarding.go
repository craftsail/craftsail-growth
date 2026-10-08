// SPDX-License-Identifier: AGPL-3.0-or-later

package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

// ProjectProgress is separate from Project so stale project saves cannot erase
// a user's confirmations or overwrite the first value event.
type ProjectProgress struct {
	ProjectID            uint64 `gorm:"primaryKey;autoIncrement:false" json:"project_id"`
	BrandRevision        string `gorm:"size:64" json:"-"`
	BrandConfirmedAt     *int64 `json:"brand_confirmed_at"`
	BrandConfirmedBy     uint64 `json:"brand_confirmed_by"`
	QuestionsRevision    string `gorm:"size:64" json:"-"`
	QuestionsConfirmedAt *int64 `json:"questions_confirmed_at"`
	QuestionsConfirmedBy uint64 `json:"questions_confirmed_by"`
	FirstValueAt         *int64 `json:"first_value_at"`
	FirstValueBy         uint64 `json:"first_value_by"`
	FirstValueKind       string `gorm:"size:32" json:"first_value_kind"`
	FirstValueRef        uint64 `json:"first_value_ref"`
}

func reviewHash(value any) string {
	b, _ := json.Marshal(value)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func BrandReviewRevision(p *Project) string {
	return reviewHash(struct {
		Version    int
		Name, Site string
		NoSite     bool
		Brand      Brand
	}{1, p.Name, p.Site, p.NoSite, p.Brand})
}
func QuestionsReviewRevision(rows []Question) string {
	clean := append([]Question{}, rows...)
	for i := range clean {
		clean[i].ID = 0
		clean[i].ProjectID = 0
		clean[i].CreatedAt = 0
		clean[i].UpdatedAt = 0
		clean[i].SystemTags = nil
		clean[i].Tags = SanitizeTags(clean[i].Tags)
	}
	sort.Slice(clean, func(i, j int) bool { return clean[i].QID < clean[j].QID })
	return reviewHash(struct {
		Version int
		Rows    []Question
	}{1, clean})
}

// IsPendingFact recognizes generated placeholders, including legacy data.
func IsPendingFact(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || s == "TBD" || s == "待确认" {
		return true
	}
	for _, prefix := range []string{"(TBD", "（TBD", "TBD:", "（待补", "(待补"} {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}
