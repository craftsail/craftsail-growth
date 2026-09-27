// SPDX-License-Identifier: AGPL-3.0-or-later

package bootstrap

import (
	"context"
	"strings"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

func TestSaveBrandCleansAndRendersFacts(t *testing.T) {
	db := bootDB(t)
	ctx := context.Background()
	p, err := project.New(db).Create(ctx, project.CreateInput{Name: "Acme", Slug: "acme", URL: "https://acme.test"})
	if err != nil {
		t.Fatal(err)
	}
	p.Brand.Offers = []model.Offer{{Name: "Pro", Price: "9"}}
	if err := db.Save(p).Error; err != nil {
		t.Fatal(err)
	}
	svc := New(db)
	md, err := svc.SaveBrand(ctx, "acme", "Acme CLI", model.Brand{
		Aliases:     []string{"acme", "待确认", " "},
		Definition:  "Acme CLI converts documents from the command line.",
		Industry:    "TBD",
		TargetUsers: "（待补：先说目标客群）",
		KeyNumbers:  []model.KeyNumber{{Fact: "Formats", Value: "12", Source: "site"}, {}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "Acme CLI converts documents") || !strings.Contains(md, "# Acme CLI · Brand facts") {
		t.Fatalf("facts card not rendered from the form:\n%s", md)
	}
	got, err := svc.Brand(ctx, "acme")
	if err != nil {
		t.Fatal(err)
	}
	b := got.Brand
	if got.Name != "Acme CLI" || len(b.Aliases) != 1 || b.Industry != "" || b.TargetUsers != "" || len(b.KeyNumbers) != 1 {
		t.Fatalf("placeholders must be dropped, got name=%q %+v", got.Name, b)
	}
	if len(b.Offers) != 1 {
		t.Fatalf("fields the form does not edit are kept, got %+v", b.Offers)
	}
}
