// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/account"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

const testToken = "test-service-token"

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := model.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

// testHandler is a handler whose API token acts as admin.
func testHandler(t *testing.T, db *gorm.DB) *Handler {
	t.Helper()
	h := NewHandler(project.New(db), nil, nil, nil, nil, testToken)
	h.accounts = account.NewForTest(db)
	return h
}

func testEngine(h *Handler) *gin.Engine {
	r := gin.New()
	Mount(r, h)
	return r
}

// call sends a JSON request. auth is testToken, a session cookie value, or "".
func call(r *gin.Engine, method, path, body, auth string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	switch {
	case auth == testToken:
		req.Header.Set("X-Craftsail-Growth-Token", auth)
	case auth != "":
		req.AddCookie(&http.Cookie{Name: sessionCookie, Value: auth})
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
