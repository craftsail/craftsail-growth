// SPDX-License-Identifier: AGPL-3.0-or-later

package audit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/pkg/httputil"
	"github.com/craftsail/craftsail-growth/internal/service/crawl"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

func TestCrawlAndAuditLocalServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/robots.txt":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("User-agent: *\nAllow: /\nSitemap: http://" + r.Host + "/sitemap.xml\n"))
		case "/sitemap.xml":
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<?xml version="1.0"?><urlset><url><loc>http://` + r.Host + `/pricing</loc></url></urlset>`))
		case "/pricing":
			_, _ = w.Write([]byte(`<!doctype html><html lang="zh"><head><title>价格</title>
<link rel="canonical" href="http://` + r.Host + `/pricing">
<script type="application/ld+json">{"@type":"Offer"}</script></head>
<body><article><h1>价格</h1><h2>套餐</h2><p>这是一套生成引擎优化工具，定义为面向品牌的诊断平台。第一步注册。第二步抓取。</p>
<p>已服务 300家 客户，覆盖 24小时 响应，增长率 20%。</p>
<ul><li>基础</li><li>专业</li><li>旗舰</li></ul></article></body></html>`))
		default:
			_, _ = w.Write([]byte(`<!doctype html><html lang="zh"><head><title>首页</title>
<link rel="canonical" href="http://` + r.Host + `/">
<script type="application/ld+json">{"@type":"Organization"}</script></head>
<body><article><h1>首页</h1><h2>产品</h2><p>这是一款生成引擎优化产品，指的是让 AI 引用品牌。</p>
<p><a href="/pricing">价格</a></p></article></body></html>`))
		}
	}))
	defer srv.Close()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := model.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	p, err := project.New(db).Create(context.Background(), project.CreateInput{URL: srv.URL, Name: "Demo"})
	if err != nil {
		t.Fatal(err)
	}
	cs := crawl.New(db, httputil.New())
	cs.Delay = 0
	res, err := cs.Run(context.Background(), p.Slug, 8)
	if err != nil {
		t.Fatal(err)
	}
	if res.PagesOK < 1 {
		t.Fatalf("pages_ok=%d crawled=%d", res.PagesOK, res.PagesCrawled)
	}
	rep, err := New(db).Run(context.Background(), p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if rep.PageCount < 1 {
		t.Fatal("no audit pages")
	}
	if len(rep.Layers) != 4 {
		t.Fatalf("layers %d", len(rep.Layers))
	}
}
