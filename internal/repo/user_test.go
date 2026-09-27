// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"testing"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func TestUsersMembersSessions(t *testing.T) {
	db := testDB(t) // existing helper in this package (see webstats_test.go)
	ctx := context.Background()
	users := &Users{DB: db}
	u := &model.User{Username: "ana", PasswordHash: "h", Role: model.RoleMember}
	if err := users.Create(ctx, u); err != nil {
		t.Fatal(err)
	}
	got, err := users.ByUsername(ctx, "ANA")
	if err != nil || got == nil || got.ID != u.ID {
		t.Fatalf("by username (case-insensitive): %v %+v", err, got)
	}

	members := &Members{DB: db}
	if err := members.Set(ctx, u.ID, 7, model.AccessView); err != nil {
		t.Fatal(err)
	}
	if err := members.Set(ctx, u.ID, 7, model.AccessEdit); err != nil {
		t.Fatal(err)
	}
	if a, _ := members.Access(ctx, u.ID, 7); a != model.AccessEdit {
		t.Fatalf("access %q", a)
	}
	if err := members.Set(ctx, u.ID, 7, ""); err != nil {
		t.Fatal(err)
	}
	if a, _ := members.Access(ctx, u.ID, 7); a != "" {
		t.Fatalf("access after revoke %q", a)
	}

	sessions := &Sessions{DB: db}
	if err := sessions.Create(ctx, &model.Session{TokenHash: "abc", UserID: u.ID, ExpiresAt: time.Now().Add(time.Hour).Unix()}); err != nil {
		t.Fatal(err)
	}
	if s, _ := sessions.Live(ctx, "abc", time.Now().Unix()); s == nil {
		t.Fatal("live session not found")
	}
	if s, _ := sessions.Live(ctx, "abc", time.Now().Add(2*time.Hour).Unix()); s != nil {
		t.Fatal("expired session returned")
	}
	if err := sessions.DeleteForUser(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	if s, _ := sessions.Live(ctx, "abc", time.Now().Unix()); s != nil {
		t.Fatal("session survived DeleteForUser")
	}
}
