// SPDX-License-Identifier: AGPL-3.0-or-later

package invoker

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ego-component/egorm/manager"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// sqliteDefaults keep a single-file database usable while background jobs
// and the dashboard write at the same time: WAL lets reads run during a
// write, busy_timeout waits instead of failing with "database is locked",
// and immediate transactions take the write lock up front so two
// transactions never deadlock upgrading from read to write.
const sqliteDefaults = "_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_txlock=immediate"

// sqliteParser lets egorm open SQLite through the pure-Go glebarez driver,
// so the binary still builds with CGO_ENABLED=0.
type sqliteParser struct{}

func init() {
	manager.Register(&sqliteParser{})
}

func (p *sqliteParser) Scheme() string { return "sqlite" }

func (p *sqliteParser) NamingStrategy() schema.Namer { return nil }

func (p *sqliteParser) GetDialector(dsn string) gorm.Dialector {
	return sqlite.Open(sqliteDSN(dsn))
}

func (p *sqliteParser) ParseDSN(dsn string) (*manager.DSN, error) {
	path := sqlitePath(dsn)
	name := filepath.Base(path)
	if path == "" || strings.Contains(dsn, ":memory:") {
		name = "memory"
	}
	return &manager.DSN{Net: "file", Addr: path, DBName: name, Params: map[string]string{}}, nil
}

// sqliteDSN appends the default pragmas unless the DSN already sets its own
// query string.
func sqliteDSN(dsn string) string {
	if strings.Contains(dsn, "?") {
		return dsn
	}
	return dsn + "?" + sqliteDefaults
}

// sqlitePath returns the file path of a DSN such as "data/app.db" or
// "file:data/app.db?cache=shared"; empty for in-memory databases.
func sqlitePath(dsn string) string {
	path := strings.TrimPrefix(dsn, "file:")
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}
	if path == "" || strings.Contains(path, ":memory:") {
		return ""
	}
	return path
}

// ensureSQLiteDir creates the directory that holds the database file, so a
// fresh checkout can start with the default dsn.
func ensureSQLiteDir(dsn string) error {
	path := sqlitePath(dsn)
	if path == "" {
		return nil
	}
	return os.MkdirAll(filepath.Dir(path), 0o700)
}
