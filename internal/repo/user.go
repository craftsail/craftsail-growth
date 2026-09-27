// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/craftsail/craftsail-growth/internal/model"
)

// ErrDuplicate marks a unique-constraint violation on insert. The app's
// *gorm.DB is not opened with TranslateError, so MySQL and SQLite each
// raise their own driver error; isDuplicate maps both to this one sentinel.
var ErrDuplicate = errors.New("duplicate row")

// isDuplicate reports whether err is a unique-constraint violation.
func isDuplicate(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "Duplicate entry") || strings.Contains(msg, "UNIQUE constraint failed")
}

type Users struct{ DB *gorm.DB }

// WithTx binds the repo to a transaction the caller already holds, so a
// check and a write can share one lock instead of racing.
func (r *Users) WithTx(tx *gorm.DB) *Users { return &Users{DB: tx} }

func (r *Users) Create(ctx context.Context, u *model.User) error {
	u.Username = strings.TrimSpace(u.Username)
	err := r.DB.WithContext(ctx).Create(u).Error
	if isDuplicate(err) {
		return ErrDuplicate
	}
	return err
}

func (r *Users) ByID(ctx context.Context, id uint64) (*model.User, error) {
	var u model.User
	err := r.DB.WithContext(ctx).First(&u, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

// ByUsername looks up a user by username. Usernames are stored lower-cased,
// so the input is lower-cased here too: "Ana" finds the row for "ana". The
// plain equality check (no SQL LOWER()) can use the unique index.
func (r *Users) ByUsername(ctx context.Context, name string) (*model.User, error) {
	var u model.User
	err := r.DB.WithContext(ctx).Where("username = ?", strings.ToLower(strings.TrimSpace(name))).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func (r *Users) List(ctx context.Context) ([]model.User, error) {
	var rows []model.User
	err := r.DB.WithContext(ctx).Order("username").Find(&rows).Error
	return rows, err
}

func (r *Users) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.DB.WithContext(ctx).Model(&model.User{}).Count(&n).Error
	return n, err
}

// SetLastLogin updates only last_login_at, so it never clobbers a change
// made to another column by a concurrent request.
func (r *Users) SetLastLogin(ctx context.Context, id uint64, ts int64) error {
	return r.DB.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).UpdateColumn("last_login_at", ts).Error
}

// SetPasswordHash updates password_hash and clears must_change_password.
func (r *Users) SetPasswordHash(ctx context.Context, id uint64, hash string) error {
	return r.DB.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).
		UpdateColumns(map[string]any{"password_hash": hash, "must_change_password": false}).Error
}

// Update sets the given columns without overwriting the rest of the row.
func (r *Users) Update(ctx context.Context, id uint64, values map[string]any) error {
	return r.DB.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Updates(values).Error
}

// DeleteTx is Delete run inside a transaction the caller already holds, so
// it can share a lock the caller took earlier in that same transaction
// (the last-admin check).
func (r *Users) DeleteTx(tx *gorm.DB, id uint64) error {
	if err := tx.Where("user_id = ?", id).Delete(&model.ProjectMember{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", id).Delete(&model.Session{}).Error; err != nil {
		return err
	}
	return tx.Delete(&model.User{}, id).Error
}

type Members struct{ DB *gorm.DB }

// Set grants view or edit, or removes the membership when access is "".
func (r *Members) Set(ctx context.Context, userID, projectID uint64, access string) error {
	db := r.DB.WithContext(ctx)
	if access == "" {
		return db.Where("user_id = ? AND project_id = ?", userID, projectID).Delete(&model.ProjectMember{}).Error
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "project_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"access", "updated_at"}),
	}).Create(&model.ProjectMember{UserID: userID, ProjectID: projectID, Access: access}).Error
}

func (r *Members) Access(ctx context.Context, userID, projectID uint64) (string, error) {
	var m model.ProjectMember
	err := r.DB.WithContext(ctx).Where("user_id = ? AND project_id = ?", userID, projectID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	return m.Access, err
}

// ForUser maps project ID to access for one user.
func (r *Members) ForUser(ctx context.Context, userID uint64) (map[uint64]string, error) {
	var rows []model.ProjectMember
	if err := r.DB.WithContext(ctx).Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint64]string, len(rows))
	for _, m := range rows {
		out[m.ProjectID] = m.Access
	}
	return out, nil
}

type Sessions struct{ DB *gorm.DB }

// WithTx binds the repo to a transaction the caller already holds.
func (r *Sessions) WithTx(tx *gorm.DB) *Sessions { return &Sessions{DB: tx} }

func (r *Sessions) Create(ctx context.Context, s *model.Session) error {
	return r.DB.WithContext(ctx).Create(s).Error
}

// Live returns the session for a token hash if it has not expired at now.
func (r *Sessions) Live(ctx context.Context, tokenHash string, now int64) (*model.Session, error) {
	var s model.Session
	err := r.DB.WithContext(ctx).Where("token_hash = ? AND expires_at > ?", tokenHash, now).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &s, err
}

func (r *Sessions) DeleteByHash(ctx context.Context, tokenHash string) error {
	return r.DB.WithContext(ctx).Where("token_hash = ?", tokenHash).Delete(&model.Session{}).Error
}

func (r *Sessions) DeleteForUser(ctx context.Context, userID uint64) error {
	return r.DB.WithContext(ctx).Where("user_id = ?", userID).Delete(&model.Session{}).Error
}

func (r *Sessions) DeleteExpired(ctx context.Context, now int64) error {
	return r.DB.WithContext(ctx).Where("expires_at <= ?", now).Delete(&model.Session{}).Error
}
