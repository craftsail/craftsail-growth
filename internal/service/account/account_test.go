// SPDX-License-Identifier: AGPL-3.0-or-later

package account

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/craftsail/craftsail-growth/internal/model"
)

const goodPass = "correct-horse-battery"

func svc(t *testing.T) *Service {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	return NewForTest(db)
}

func TestCreateAndSignIn(t *testing.T) {
	s, ctx := svc(t), context.Background()
	u, err := s.CreateUser(ctx, "ana", goodPass, model.RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	if u.PasswordHash == goodPass {
		t.Fatal("password stored in clear text")
	}
	if _, err := s.SignIn(ctx, "ana", "wrong-password-here"); !errors.Is(err, ErrBadCredentials) {
		t.Fatalf("wrong password: %v", err)
	}
	tok, err := s.SignIn(ctx, "ANA", goodPass)
	if err != nil || tok == "" {
		t.Fatalf("sign in: %v", err)
	}
	got, err := s.UserForToken(ctx, tok)
	if err != nil || got == nil || got.ID != u.ID {
		t.Fatalf("user for token: %v %+v", err, got)
	}
}

func TestSignInUnknownUsername(t *testing.T) {
	s, ctx := svc(t), context.Background()
	if _, err := s.CreateUser(ctx, "ana", goodPass, model.RoleMember); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SignIn(ctx, "nobody", goodPass); !errors.Is(err, ErrBadCredentials) {
		t.Fatalf("unknown username: %v", err)
	}
}

func TestCreateUserCaseInsensitiveDuplicate(t *testing.T) {
	s, ctx := svc(t), context.Background()
	if _, err := s.CreateUser(ctx, "Ana", goodPass, model.RoleMember); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateUser(ctx, "ana", goodPass, model.RoleMember); !errors.Is(err, ErrUserExists) {
		t.Fatalf("duplicate username: %v", err)
	}
	if _, err := s.SignIn(ctx, "ANA", goodPass); err != nil {
		t.Fatalf("sign in with different case: %v", err)
	}
}

func TestPasswordRules(t *testing.T) {
	s, ctx := svc(t), context.Background()
	for _, p := range []string{"short", "password-password", "ana"} {
		if _, err := s.CreateUser(ctx, "ana", p, model.RoleMember); err == nil {
			t.Fatalf("accepted weak password %q", p)
		}
	}
}

func TestPasswordTooLongRejected(t *testing.T) {
	s, ctx := svc(t), context.Background()
	long := strings.Repeat("a", 73)
	if _, err := s.CreateUser(ctx, "ana", long, model.RoleMember); err == nil {
		t.Fatal("accepted password over 72 bytes")
	}
}

func TestPasswordChangeEndsSessions(t *testing.T) {
	s, ctx := svc(t), context.Background()
	u, err := s.CreateUser(ctx, "ana", goodPass, model.RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	tok, err := s.SignIn(ctx, "ana", goodPass)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetPassword(ctx, u.ID, "another-long-passphrase"); err != nil {
		t.Fatal(err)
	}
	got, err := s.UserForToken(ctx, tok)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("old session still valid after password change")
	}
}

func TestDisabledUserCannotUseSession(t *testing.T) {
	s, ctx := svc(t), context.Background()
	u, err := s.CreateUser(ctx, "ana", goodPass, model.RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	tok, err := s.SignIn(ctx, "ana", goodPass)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Update(ctx, u.ID, Patch{Disabled: ptr(true)}); err != nil {
		t.Fatal(err)
	}
	got, err := s.UserForToken(ctx, tok)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("disabled user still signed in")
	}
	if _, err := s.SignIn(ctx, "ana", goodPass); !errors.Is(err, ErrBadCredentials) {
		t.Fatalf("disabled sign in: %v", err)
	}
}

func TestExpiredSession(t *testing.T) {
	s, ctx := svc(t), context.Background()
	if _, err := s.CreateUser(ctx, "ana", goodPass, model.RoleMember); err != nil {
		t.Fatal(err)
	}
	tok, err := s.SignIn(ctx, "ana", goodPass)
	if err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return time.Now().Add(SessionTTL + time.Minute) }
	got, err := s.UserForToken(ctx, tok)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("expired session accepted")
	}
}

func TestAccess(t *testing.T) {
	s, ctx := svc(t), context.Background()
	admin, err := s.CreateUser(ctx, "root", goodPass, model.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	ana, err := s.CreateUser(ctx, "ana", goodPass, model.RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.Access(ctx, admin, 1)
	if err != nil {
		t.Fatal(err)
	}
	if a != model.AccessEdit {
		t.Fatalf("admin access %q", a)
	}
	a, err = s.Access(ctx, ana, 1)
	if err != nil {
		t.Fatal(err)
	}
	if a != "" {
		t.Fatalf("unshared access %q", a)
	}
	if err := s.Grant(ctx, ana.ID, 1, model.AccessView); err != nil {
		t.Fatal(err)
	}
	a, err = s.Access(ctx, ana, 1)
	if err != nil {
		t.Fatal(err)
	}
	if a != model.AccessView {
		t.Fatalf("view access %q", a)
	}
	if err := s.Grant(ctx, ana.ID, 1, "owner"); err == nil {
		t.Fatal("accepted unknown access level")
	}
}

func TestGrantEmptyRevokes(t *testing.T) {
	s, ctx := svc(t), context.Background()
	ana, err := s.CreateUser(ctx, "ana", goodPass, model.RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Grant(ctx, ana.ID, 1, model.AccessView); err != nil {
		t.Fatal(err)
	}
	if err := s.Grant(ctx, ana.ID, 1, ""); err != nil {
		t.Fatal(err)
	}
	a, err := s.Access(ctx, ana, 1)
	if err != nil {
		t.Fatal(err)
	}
	if a != "" {
		t.Fatalf("access after revoke %q", a)
	}
}

func TestLastAdminIsProtected(t *testing.T) {
	s, ctx := svc(t), context.Background()
	root, err := s.CreateUser(ctx, "root", goodPass, model.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Update(ctx, root.ID, Patch{Role: ptr(model.RoleMember)}); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("demote last admin: %v", err)
	}
	if err := s.Update(ctx, root.ID, Patch{Disabled: ptr(true)}); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("disable last admin: %v", err)
	}
	if err := s.Delete(ctx, root.ID); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("delete last admin: %v", err)
	}
}

func TestUpdateInvalidRole(t *testing.T) {
	s, ctx := svc(t), context.Background()
	ana, err := s.CreateUser(ctx, "ana", goodPass, model.RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Update(ctx, ana.ID, Patch{Role: ptr("owner")}); !errors.Is(err, ErrBadRole) {
		t.Fatalf("invalid role: %v", err)
	}
}

func TestUpdateValidatesRoleBeforeLastAdminCheck(t *testing.T) {
	s, ctx := svc(t), context.Background()
	root, err := s.CreateUser(ctx, "root", goodPass, model.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Update(ctx, root.ID, Patch{Role: ptr("owner")}); !errors.Is(err, ErrBadRole) {
		t.Fatalf("bad role on the last admin should be ErrBadRole, got %v", err)
	}
}

func TestDisabledAdminDoesNotCount(t *testing.T) {
	s, ctx := svc(t), context.Background()
	root, err := s.CreateUser(ctx, "root", goodPass, model.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	other, err := s.CreateUser(ctx, "other", goodPass, model.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Update(ctx, other.ID, Patch{Disabled: ptr(true)}); err != nil {
		t.Fatal(err)
	}
	if err := s.Update(ctx, root.ID, Patch{Role: ptr(model.RoleMember)}); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("demote the only enabled admin: %v", err)
	}
}

func TestDemoteOneOfTwoAdminsSucceeds(t *testing.T) {
	s, ctx := svc(t), context.Background()
	root, err := s.CreateUser(ctx, "root", goodPass, model.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateUser(ctx, "other", goodPass, model.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if err := s.Update(ctx, root.ID, Patch{Role: ptr(model.RoleMember)}); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteRemovesMembershipsAndSessions(t *testing.T) {
	s, ctx := svc(t), context.Background()
	ana, err := s.CreateUser(ctx, "ana", goodPass, model.RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Grant(ctx, ana.ID, 1, model.AccessView); err != nil {
		t.Fatal(err)
	}
	tok, err := s.SignIn(ctx, "ana", goodPass)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, ana.ID); err != nil {
		t.Fatal(err)
	}
	if a, err := s.members.Access(ctx, ana.ID, 1); err != nil || a != "" {
		t.Fatalf("membership survived delete: %v %q", err, a)
	}
	got, err := s.UserForToken(ctx, tok)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("session survived delete")
	}
}

func TestSignOutEndsSession(t *testing.T) {
	s, ctx := svc(t), context.Background()
	if _, err := s.CreateUser(ctx, "ana", goodPass, model.RoleMember); err != nil {
		t.Fatal(err)
	}
	tok, err := s.SignIn(ctx, "ana", goodPass)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SignOut(ctx, tok); err != nil {
		t.Fatal(err)
	}
	got, err := s.UserForToken(ctx, tok)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("session survived sign out")
	}
}

func TestSeedAdminFromEnvOnlyOnce(t *testing.T) {
	s, ctx := svc(t), context.Background()
	if err := s.SeedAdmin(ctx, "root", goodPass); err != nil {
		t.Fatal(err)
	}
	if err := s.SeedAdmin(ctx, "other", goodPass); err != nil {
		t.Fatal(err)
	}
	users, err := s.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].Username != "root" || users[0].Role != model.RoleAdmin {
		t.Fatalf("seeded %+v", users)
	}
}

func TestSeedAdminEmptyPasswordNoop(t *testing.T) {
	s, ctx := svc(t), context.Background()
	if err := s.SeedAdmin(ctx, "root", ""); err != nil {
		t.Fatal(err)
	}
	users, err := s.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 0 {
		t.Fatalf("seeded with empty password: %+v", users)
	}
}

func ptr[T any](v T) *T { return &v }

func TestNewPanicsOnInvalidCost(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on invalid bcrypt cost")
		}
	}()
	newService(nil, bcrypt.MaxCost+1)
}

func TestUsernameAllowsAtSign(t *testing.T) {
	s, ctx := svc(t), context.Background()
	u, err := s.CreateUser(ctx, "ana@example.com", goodPass, model.RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	if u.Username != "ana@example.com" {
		t.Fatalf("username %q", u.Username)
	}
	if _, err := s.SignIn(ctx, "ANA@EXAMPLE.COM", goodPass); err != nil {
		t.Fatalf("sign in: %v", err)
	}
}

func TestSeedAdminValidates(t *testing.T) {
	s, ctx := svc(t), context.Background()
	if err := s.SeedAdmin(ctx, "Admin User", goodPass); !IsValidation(err) {
		t.Fatalf("a username with a space must be rejected: %v", err)
	}
	if err := s.SeedAdmin(ctx, "root", "short"); !IsValidation(err) {
		t.Fatalf("a weak password must be rejected: %v", err)
	}
	if empty, _ := s.NeedsSetup(ctx); !empty {
		t.Fatal("a rejected seed must not create a user")
	}
	if err := s.SeedAdmin(ctx, "", goodPass); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SignIn(ctx, DefaultUsername, goodPass); err != nil {
		t.Fatalf("an empty username seeds %q: %v", DefaultUsername, err)
	}
}

func TestIsValidation(t *testing.T) {
	s, ctx := svc(t), context.Background()
	if _, err := s.CreateUser(ctx, "!!bad!!", goodPass, model.RoleMember); !IsValidation(err) {
		t.Fatalf("bad username should be a validation error: %v", err)
	}
	if _, err := s.CreateUser(ctx, "ana", "too-short", model.RoleMember); !IsValidation(err) {
		t.Fatalf("weak password should be a validation error: %v", err)
	}
	if _, err := s.CreateUser(ctx, "ana", goodPass, model.RoleMember); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateUser(ctx, "ana", goodPass, model.RoleMember); !IsValidation(err) {
		t.Fatalf("duplicate username should be a validation error: %v", err)
	}
	if IsValidation(errors.New("boom")) {
		t.Fatal("an ordinary error must not look like a validation error")
	}
}
