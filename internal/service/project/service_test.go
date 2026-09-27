// SPDX-License-Identifier: AGPL-3.0-or-later

package project

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := model.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCreateNoSiteRequiresName(t *testing.T) {
	svc := New(testDB(t))
	_, err := svc.Create(context.Background(), CreateInput{NoSite: true})
	if !errors.Is(err, ErrNameRequired) {
		t.Fatalf("err = %v, want ErrNameRequired", err)
	}
}

func TestCreateDerivesSlugFromHost(t *testing.T) {
	svc := New(testDB(t))
	p, err := svc.Create(context.Background(), CreateInput{
		URL: "https://Acme-Tools.io",
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Slug != "acme-tools" {
		t.Fatalf("slug = %q", p.Slug)
	}
	if p.Site != "https://Acme-Tools.io" {
		t.Fatalf("site = %q", p.Site)
	}
	if p.Market != model.MarketAll || len(p.Platforms) != len(model.DefaultPlatforms()) {
		t.Fatalf("market/platforms = %s %#v", p.Market, p.Platforms)
	}
}

func TestCreateDuplicateSlugRejected(t *testing.T) {
	svc := New(testDB(t))
	in := CreateInput{URL: "https://acme.example", Name: "Acme"}
	if _, err := svc.Create(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Create(context.Background(), in)
	if !errors.Is(err, ErrSlugTaken) {
		t.Fatalf("err = %v, want ErrSlugTaken", err)
	}
}

func TestCreateForceReplaces(t *testing.T) {
	svc := New(testDB(t))
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{Name: "A", Slug: "acme", NoSite: true}); err != nil {
		t.Fatal(err)
	}
	p, err := svc.Create(ctx, CreateInput{Name: "B", Slug: "acme", NoSite: true, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "B" {
		t.Fatalf("got %+v", p)
	}
	list, err := svc.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d", len(list))
	}
}

func TestUpdateSiteKeepsSlug(t *testing.T) {
	svc := New(testDB(t))
	ctx := context.Background()
	p, err := svc.Create(ctx, CreateInput{URL: "https://wrong.example", Name: "甲工"})
	if err != nil {
		t.Fatal(err)
	}
	site := "https://right.example"
	name := "甲工正名"
	got, err := svc.Update(ctx, p.Slug, UpdateInput{URL: &site, Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	if got.Slug != p.Slug {
		t.Fatalf("slug changed to %s", got.Slug)
	}
	if got.Site != site || got.Name != name || got.NoSite {
		t.Fatalf("%+v", got)
	}
}

func TestUpdateNoSiteClearsURL(t *testing.T) {
	svc := New(testDB(t))
	ctx := context.Background()
	p, err := svc.Create(ctx, CreateInput{URL: "https://gone.example", Name: "甲工"})
	if err != nil {
		t.Fatal(err)
	}
	no := true
	got, err := svc.Update(ctx, p.Slug, UpdateInput{NoSite: &no})
	if err != nil {
		t.Fatal(err)
	}
	if !got.NoSite || got.Site != "" {
		t.Fatalf("%+v", got)
	}
}

func TestUpdateGscAndGA4(t *testing.T) {
	svc := New(testDB(t))
	ctx := context.Background()
	p, err := svc.Create(ctx, CreateInput{Name: "Acme", Slug: "acme", NoSite: true})
	if err != nil {
		t.Fatal(err)
	}
	gsc := "sc-domain:acme.com"
	ga := "properties/123456"
	got, err := svc.Update(ctx, p.Slug, UpdateInput{GscSite: &gsc, GA4Property: &ga})
	if err != nil {
		t.Fatal(err)
	}
	if got.GscSite != gsc || got.GA4Property != "123456" {
		t.Fatalf("%#v", got)
	}
}

func TestIntentOf(t *testing.T) {
	if model.IntentOf("品牌验证") != "probe" {
		t.Fatal("probe")
	}
	if model.IntentOf("场景") != "educate" || model.IntentOf("风险") != "educate" {
		t.Fatal("educate")
	}
	if model.IntentOf("推荐") != "buyer" {
		t.Fatal("buyer")
	}
}

func TestMigrateWebstatsTables(t *testing.T) {
	db := testDB(t)
	if err := db.Exec("SELECT gsc_site, ga4_property, monitor_runs_per_day FROM projects").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("SELECT property, key_hash, query, page FROM gsc_facts").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("SELECT property, clicks, impressions FROM gsc_dailies").Error; err != nil {
		t.Fatal(err)
	}
}

func TestMigrateGoogleFactTables(t *testing.T) {
	db := testDB(t)
	for _, table := range []string{"gsc_facts", "ga_facts", "google_raws", "web_sync_states"} {
		if err := db.Exec("SELECT count(*) FROM " + table).Error; err != nil {
			t.Fatal(table, err)
		}
	}
}
