// SPDX-License-Identifier: AGPL-3.0-or-later

// Package account owns users, passwords, sessions and the project access rule.
package account

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
)

// The account created on first start when no user exists. Its password is
// public, so the API refuses everything but a password change until it is
// replaced.
const (
	DefaultUsername = "admin"
	DefaultPassword = "craftsailgrowth"
)

// SessionTTL is how long a sign-in lasts.
const SessionTTL = 30 * 24 * time.Hour

var (
	ErrBadCredentials = errors.New("wrong username or password")
	ErrUserExists     = errors.New("that username is taken")
	ErrUserNotFound   = errors.New("user not found")
	ErrLastAdmin      = errors.New("keep at least one enabled admin")
	ErrBadRole        = errors.New("role must be admin or member")
	ErrBadAccess      = errors.New("access must be view, edit or none")
	ErrBadUsername    = errors.New("username: 2 to 64 letters, digits, dot, dash, underscore or @")
)

var usernameRe = regexp.MustCompile(`^[A-Za-z0-9._@-]{2,64}$`)

// ValidationError marks a rejection caused by bad input -- a password that
// breaks the rules, in practice -- rather than a system failure, so API
// handlers can map it to 400 instead of 500.
type ValidationError struct{ msg string }

func (e *ValidationError) Error() string { return e.msg }

func validationErrorf(format string, args ...any) error {
	return &ValidationError{msg: fmt.Sprintf(format, args...)}
}

// IsValidation reports whether err was caused by bad input -- an invalid
// username, role or password -- rather than a system failure such as a
// database error, so callers can choose 400 over 500.
func IsValidation(err error) bool {
	var v *ValidationError
	return errors.As(err, &v) ||
		errors.Is(err, ErrBadUsername) || errors.Is(err, ErrUserExists) || errors.Is(err, ErrBadRole)
}

type Service struct {
	users    *repo.Users
	members  *repo.Members
	sessions *repo.Sessions
	cost     int
	now      func() time.Time

	dummyHash []byte
}

func New(db *gorm.DB) *Service {
	return newService(db, 12)
}

// NewForTest is New with the cheapest bcrypt cost. Only for tests.
func NewForTest(db *gorm.DB) *Service {
	return newService(db, bcrypt.MinCost)
}

// newService builds a Service at the given bcrypt cost, generating the
// dummy hash used by SignIn's constant-time comparison up front: it must
// exist before the first request, and a failure here means bcrypt itself is
// broken, which nothing can recover from.
func newService(db *gorm.DB, cost int) *Service {
	hash, err := bcrypt.GenerateFromPassword([]byte("no-such-user-dummy-password"), cost)
	if err != nil {
		panic(err)
	}
	return &Service{
		users: &repo.Users{DB: db}, members: &repo.Members{DB: db}, sessions: &repo.Sessions{DB: db},
		cost: cost, now: time.Now, dummyHash: hash,
	}
}

// Patch changes a user; nil fields stay as they are.
type Patch struct {
	Role     *string
	Disabled *bool
}

func validatePassword(user, pass string) error {
	if len(pass) > 72 {
		return validationErrorf("password must be at most 72 bytes")
	}
	if len([]rune(pass)) < 12 {
		return validationErrorf("password must be at least 12 characters")
	}
	low := strings.ToLower(pass)
	for _, w := range []string{"admin", "password", "12345678", "123456789", "qwerty"} {
		if strings.Contains(low, w) {
			return validationErrorf("password is too common; choose one that is hard to guess")
		}
	}
	if strings.EqualFold(user, pass) {
		return validationErrorf("password must differ from the username")
	}
	if pass == DefaultPassword {
		return validationErrorf("choose a password other than the default one")
	}
	return nil
}

func (s *Service) NeedsSetup(ctx context.Context) (bool, error) {
	n, err := s.users.Count(ctx)
	return n == 0, err
}

func (s *Service) CreateUser(ctx context.Context, username, password, role string) (*model.User, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if !usernameRe.MatchString(username) {
		return nil, ErrBadUsername
	}
	if role != model.RoleAdmin && role != model.RoleMember {
		return nil, ErrBadRole
	}
	if err := validatePassword(username, password); err != nil {
		return nil, err
	}
	if u, err := s.users.ByUsername(ctx, username); err != nil {
		return nil, err
	} else if u != nil {
		return nil, ErrUserExists
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	if err != nil {
		return nil, err
	}
	u := &model.User{Username: username, PasswordHash: string(hash), Role: role}
	if err := s.users.Create(ctx, u); err != nil {
		if errors.Is(err, repo.ErrDuplicate) {
			return nil, ErrUserExists
		}
		return nil, err
	}
	return u, nil
}

// SeedAdmin creates the first admin from CRAFTSAIL_GROWTH_USER and
// CRAFTSAIL_GROWTH_PASSWORD. It does nothing without a password or once any
// user exists. The same username and password rules as CreateUser apply.
func (s *Service) SeedAdmin(ctx context.Context, username, password string) error {
	if password == "" {
		return nil
	}
	if strings.TrimSpace(username) == "" {
		username = DefaultUsername
	}
	if empty, err := s.NeedsSetup(ctx); err != nil || !empty {
		return err
	}
	_, err := s.CreateUser(ctx, username, password, model.RoleAdmin)
	return err
}

// SeedDefault creates admin / craftsailgrowth when there is no user at all,
// flagged so the dashboard asks for a new password.
func (s *Service) SeedDefault(ctx context.Context) error {
	if empty, err := s.NeedsSetup(ctx); err != nil || !empty {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(DefaultPassword), s.cost)
	if err != nil {
		return err
	}
	u := &model.User{Username: DefaultUsername, PasswordHash: string(hash), Role: model.RoleAdmin, MustChangePassword: true}
	if err := s.users.Create(ctx, u); err != nil && !errors.Is(err, repo.ErrDuplicate) {
		return err
	}
	return nil
}

// SignIn checks the password and returns a new session token for the
// cookie. It always runs a bcrypt comparison, whether or not the username
// exists, and only looks at Disabled after that comparison, so the time a
// request takes does not reveal which usernames are registered.
func (s *Service) SignIn(ctx context.Context, username, password string) (string, error) {
	u, err := s.users.ByUsername(ctx, username)
	if err != nil {
		return "", err
	}
	hash := s.dummyHash
	if u != nil {
		hash = []byte(u.PasswordHash)
	}
	ok := bcrypt.CompareHashAndPassword(hash, []byte(password)) == nil
	if u == nil || !ok || u.Disabled {
		return "", ErrBadCredentials
	}
	_ = s.users.SetLastLogin(ctx, u.ID, s.now().Unix())
	return s.newSession(ctx, u.ID)
}

// StartSession signs a known user in without a password, right after setup.
func (s *Service) StartSession(ctx context.Context, userID uint64) (string, error) {
	return s.newSession(ctx, userID)
}

func (s *Service) newSession(ctx context.Context, userID uint64) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	tok := hex.EncodeToString(raw)
	_ = s.sessions.DeleteExpired(ctx, s.now().Unix())
	err := s.sessions.Create(ctx, &model.Session{TokenHash: hashToken(tok), UserID: userID, ExpiresAt: s.now().Add(SessionTTL).Unix()})
	return tok, err
}

// UserForToken returns the signed-in user, or nil for an unknown, expired
// or disabled session.
func (s *Service) UserForToken(ctx context.Context, tok string) (*model.User, error) {
	if tok == "" {
		return nil, nil
	}
	sess, err := s.sessions.Live(ctx, hashToken(tok), s.now().Unix())
	if err != nil || sess == nil {
		return nil, err
	}
	u, err := s.users.ByID(ctx, sess.UserID)
	if err != nil || u == nil || u.Disabled {
		return nil, err
	}
	return u, nil
}

func (s *Service) SignOut(ctx context.Context, tok string) error {
	return s.sessions.DeleteByHash(ctx, hashToken(tok))
}

func (s *Service) SetPassword(ctx context.Context, userID uint64, password string) error {
	u, err := s.mustUser(ctx, userID)
	if err != nil {
		return err
	}
	if err := validatePassword(u.Username, password); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	if err != nil {
		return err
	}
	if err := s.users.SetPasswordHash(ctx, userID, string(hash)); err != nil {
		return err
	}
	return s.sessions.DeleteForUser(ctx, userID)
}

// CheckPassword is used when a user changes their own password.
func (s *Service) CheckPassword(ctx context.Context, userID uint64, password string) error {
	u, err := s.mustUser(ctx, userID)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return ErrBadCredentials
	}
	return nil
}

// Update changes role and/or disabled state. When the change would take
// the last enabled admin off the role or disable them, the check and the
// write run in one transaction with a row lock, so two concurrent requests
// cannot both see an admin to spare and both proceed.
func (s *Service) Update(ctx context.Context, userID uint64, p Patch) error {
	if p.Role != nil && *p.Role != model.RoleAdmin && *p.Role != model.RoleMember {
		return ErrBadRole
	}
	return s.users.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		users := s.users.WithTx(tx)
		u, err := users.ByID(ctx, userID)
		if err != nil {
			return err
		}
		if u == nil {
			return fmt.Errorf("%w: %d", ErrUserNotFound, userID)
		}
		losesAdmin := u.Role == model.RoleAdmin && !u.Disabled &&
			((p.Role != nil && *p.Role != model.RoleAdmin) || (p.Disabled != nil && *p.Disabled))
		if losesAdmin {
			if err := keepOneAdminLocked(tx); err != nil {
				return err
			}
		}
		values := map[string]any{}
		if p.Role != nil {
			values["role"] = *p.Role
		}
		disabledNow := u.Disabled
		if p.Disabled != nil {
			values["disabled"] = *p.Disabled
			disabledNow = *p.Disabled
		}
		if len(values) > 0 {
			if err := users.Update(ctx, userID, values); err != nil {
				return err
			}
		}
		// Admins need no memberships; dropping them on promotion means a
		// later demotion starts from nothing instead of reviving old access.
		if p.Role != nil && *p.Role == model.RoleAdmin && u.Role != model.RoleAdmin {
			if err := tx.Where("user_id = ?", userID).Delete(&model.ProjectMember{}).Error; err != nil {
				return err
			}
		}
		if disabledNow {
			return s.sessions.WithTx(tx).DeleteForUser(ctx, userID)
		}
		return nil
	})
}

// Delete removes a user. Removing the last enabled admin is refused; the
// check and the delete run in one transaction with a row lock so two
// concurrent deletes cannot both pass the check.
func (s *Service) Delete(ctx context.Context, userID uint64) error {
	return s.users.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		users := s.users.WithTx(tx)
		u, err := users.ByID(ctx, userID)
		if err != nil {
			return err
		}
		if u == nil {
			return fmt.Errorf("%w: %d", ErrUserNotFound, userID)
		}
		if u.Role == model.RoleAdmin && !u.Disabled {
			if err := keepOneAdminLocked(tx); err != nil {
				return err
			}
		}
		return users.DeleteTx(tx, userID)
	})
}

// keepOneAdminLocked counts enabled admins with a row lock held for the
// rest of the caller's transaction, so a concurrent demote, disable or
// delete of another admin cannot commit until this transaction is done.
func keepOneAdminLocked(tx *gorm.DB) error {
	var n int64
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&model.User{}).
		Where("role = ? AND disabled = ?", model.RoleAdmin, false).
		Count(&n).Error
	if err != nil {
		return err
	}
	if n <= 1 {
		return ErrLastAdmin
	}
	return nil
}

func (s *Service) List(ctx context.Context) ([]model.User, error) { return s.users.List(ctx) }

func (s *Service) ByUsername(ctx context.Context, name string) (*model.User, error) {
	u, err := s.users.ByUsername(ctx, name)
	if err == nil && u == nil {
		err = ErrUserNotFound
	}
	return u, err
}

// Grant sets a member's access to a project: view, edit, or "" for none.
func (s *Service) Grant(ctx context.Context, userID, projectID uint64, access string) error {
	if access != "" && access != model.AccessView && access != model.AccessEdit {
		return ErrBadAccess
	}
	if _, err := s.mustUser(ctx, userID); err != nil {
		return err
	}
	return s.members.Set(ctx, userID, projectID, access)
}

// Access is the one access rule: admins edit everything, members get their
// membership row, everyone else gets "". Callers must pass a user obtained
// from UserForToken or SignIn: those never return a disabled user, so
// Access does not check Disabled itself.
func (s *Service) Access(ctx context.Context, u *model.User, projectID uint64) (string, error) {
	if u == nil {
		return "", nil
	}
	if u.Role == model.RoleAdmin {
		return model.AccessEdit, nil
	}
	return s.members.Access(ctx, u.ID, projectID)
}

// Grants maps project ID to access for a member. An admin may have
// leftover membership rows from before they were promoted; Access ignores
// them for an admin, and callers of Grants should too.
func (s *Service) Grants(ctx context.Context, userID uint64) (map[uint64]string, error) {
	return s.members.ForUser(ctx, userID)
}

func (s *Service) mustUser(ctx context.Context, id uint64) (*model.User, error) {
	u, err := s.users.ByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, fmt.Errorf("%w: %d", ErrUserNotFound, id)
	}
	return u, nil
}

func hashToken(tok string) string {
	sum := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(sum[:])
}
