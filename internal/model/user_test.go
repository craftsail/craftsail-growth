// SPDX-License-Identifier: AGPL-3.0-or-later

package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestUserTablesMigrate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	for _, m := range []any{&User{}, &ProjectMember{}, &Session{}} {
		if !db.Migrator().HasTable(m) {
			t.Fatalf("missing table for %T", m)
		}
	}
	u := User{Username: "ana", PasswordHash: "x", Role: RoleMember}
	if err := db.Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&User{Username: "ana", PasswordHash: "y", Role: RoleMember}).Error; err == nil {
		t.Fatal("duplicate username accepted")
	}
	m := ProjectMember{UserID: u.ID, ProjectID: 1, Access: AccessView}
	if err := db.Create(&m).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&ProjectMember{UserID: u.ID, ProjectID: 1, Access: AccessEdit}).Error; err == nil {
		t.Fatal("duplicate membership accepted")
	}
}
