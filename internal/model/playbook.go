// SPDX-License-Identifier: AGPL-3.0-or-later

package model

// PlaybookConfirmation records that a user checked a pass signal by hand,
// for signals that no Google API reports. One row per project and signal.
type PlaybookConfirmation struct {
	ProjectID   uint64 `gorm:"primaryKey;autoIncrement:false" json:"project_id"`
	Signal      string `gorm:"primaryKey;size:32" json:"signal"`
	ConfirmedAt int64  `json:"confirmed_at"`
	ConfirmedBy uint64 `json:"confirmed_by"`
}
