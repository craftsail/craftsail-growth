// SPDX-License-Identifier: AGPL-3.0-or-later

package model

// QuestionLibrary preserves the exact questions and targeting configuration
// used by a sampling run. Legacy samples retain their empty revision.
type QuestionLibrary struct {
	ID        uint64     `gorm:"primaryKey" json:"id"`
	ProjectID uint64     `gorm:"uniqueIndex:uk_question_library;not null" json:"project_id"`
	Revision  string     `gorm:"size:64;uniqueIndex:uk_question_library;not null" json:"revision"`
	Language  string     `gorm:"size:8" json:"language"`
	Region    string     `gorm:"size:64" json:"region"`
	Questions []Question `gorm:"serializer:json;type:longtext" json:"questions"`
	CreatedAt int64      `json:"created_at"`
}

func LibraryRevision(rows []Question, language, region string) string {
	return RowKey("library-v1", QuestionsReviewRevision(rows), language, region)
}
