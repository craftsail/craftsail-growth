// SPDX-License-Identifier: AGPL-3.0-or-later

package invoker

import (
	"fmt"

	"github.com/ego-component/egorm"
	"github.com/gotomicro/ego/core/econf"
	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
)

var DB *gorm.DB

func Init() error {
	if econf.GetString("db.dialect") == "sqlite" {
		if err := ensureSQLiteDir(econf.GetString("db.dsn")); err != nil {
			return fmt.Errorf("sqlite: %w", err)
		}
	}
	DB = egorm.Load("db").Build()
	if DB == nil {
		return fmt.Errorf("database is not open; check [db] in config/default.toml")
	}
	tune(DB)
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping: %w (check [db] dsn; for MySQL create the database with deploy/compose.dev-mysql.yml or deploy/init.sql)", err)
	}
	return nil
}

// tune splits bulk inserts so a wide row type such as Sample stays under
// SQLite's 32766 bound parameters (MySQL allows 65535).
func tune(db *gorm.DB) {
	db.CreateBatchSize = 500
}

func Migrate() error {
	if DB == nil {
		if err := Init(); err != nil {
			return err
		}
	}
	return model.AutoMigrate(DB)
}
