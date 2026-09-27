// SPDX-License-Identifier: AGPL-3.0-or-later

package bootstrap

import (
	"fmt"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
)

type BrandFacts struct {
	Name           string
	Aliases        []string
	Products       []string
	Industry       string
	TargetUsers    string
	BusinessGoal   string
	Definition     string
	KeyNumbers     []model.KeyNumber
	Suitable       []string
	Unsuitable     []string
	Disambiguation []string
	Pricing        []model.Offer
	Uncertain      []string
}

// Pending marks a fact the site does not state. The legacy value "待确认"
// is still recognized in stored data.
const Pending = "TBD"

func isPending(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || s == Pending || s == "待确认" {
		return true
	}
	// Template placeholders: "(TBD: …)" and the legacy "（待补：…）".
	for _, p := range []string{"(TBD", "（TBD", "TBD:", "（待补", "(待补"} {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func RenderFacts(brand BrandFacts, site string, comps []model.Competitor) string {
	name := orPending(brand.Name)
	L := []string{
		fmt.Sprintf("# %s · Brand facts", brand.Name),
		"",
		fmt.Sprintf("> Extracted from the site text by `bootstrap` on %s. **Review every line.**", time.Now().Format("2006-01-02")),
		"> Evidence: `A confirmed by the brand` / `B confirmed by a third party` / `C internal, not yet approved` / `D needs proof` / `E do not use`.",
		"> Fields marked TBD are not stated on the site. **Fill them in or decide not to say them publicly.**",
		"",
		"## Entity",
		"",
		"| Item | Value | Evidence |",
		"|---|---|---|",
		fmt.Sprintf("| Canonical name | %s | A site |", name),
		fmt.Sprintf("| Aliases | %s | A site |", orJoin(brand.Aliases)),
		fmt.Sprintf("| Website | %s | A |", orPending(site)),
		fmt.Sprintf("| Category | %s | A site |", orPending(brand.Industry)),
	}
	for _, u := range brand.Uncertain {
		L = append(L, fmt.Sprintf("| %s | **TBD** | D needs proof |", u))
	}
	def := brand.Definition
	if isPending(def) {
		def = "(TBD: start with the brand name, then the audience and the category)"
	}
	L = append(L, "", "## One-line definition", "", "> "+def, "",
		"**Use exactly this sentence in four places**: the home page hero, the about page, the JSON-LD `description` and `llms.txt`.",
		"Inconsistent wording is the most common reason AI descriptions of a brand drift.", "")
	if len(brand.Disambiguation) > 0 {
		L = append(L, "## Disambiguation", "", "Keep these statements identical on the site, in encyclopedias and in outside materials:", "")
		for i, d := range brand.Disambiguation {
			L = append(L, fmt.Sprintf("%d. %s", i+1, d))
		}
		L = append(L, "")
	}
	L = append(L, "## Key numbers", "", "| Fact | Value | Source | Evidence |", "|---|---|---|---|")
	if len(brand.KeyNumbers) == 0 {
		L = append(L, "| (no usable numbers found on the site; add them by hand) | TBD | - | D |")
	} else {
		for _, n := range brand.KeyNumbers {
			src := n.Source
			if src == "" {
				src = "site"
			}
			L = append(L, fmt.Sprintf("| %s | %s | %s | A |", n.Fact, n.Value, src))
		}
	}
	L = append(L, "")
	if len(brand.Pricing) > 0 {
		L = append(L, "## Pricing", "", "| Plan | Price | Includes |", "|---|---|---|")
		for _, p := range brand.Pricing {
			L = append(L, fmt.Sprintf("| %s | %s %s | %s |", p.Name, p.Price, p.Currency, p.Desc))
		}
		L = append(L, "")
	}
	suit := brand.Suitable
	if len(suit) == 0 {
		suit = []string{Pending}
	}
	unsuit := brand.Unsuitable
	if len(unsuit) == 0 {
		unsuit = []string{Pending}
	}
	L = append(L, "## Fit", "", "**Good fit**:", "")
	for _, x := range suit {
		L = append(L, "- "+x)
	}
	L = append(L, "", "**Not a fit**:", "")
	for _, x := range unsuit {
		L = append(L, "- "+x)
	}
	L = append(L, "", "> Saying who the product is not for makes the page more credible; one-sided praise reads as marketing copy.", "")
	if len(comps) > 0 {
		L = append(L, "## Competitors", "", "| Competitor | Status |", "|---|---|")
		for _, c := range comps {
			status := "seen in sampled answers"
			if !model.CompetitorConfirmed(c) {
				status = "**not yet seen in answers**"
			}
			L = append(L, fmt.Sprintf("| %s | %s |", c.Name, status))
		}
		L = append(L, "> Competitors not yet seen in answers are LLM suggestions. Do not name them in public content.", "")
	}
	L = append(L,
		"## Do not say", "",
		"- Customer names, figures or credentials that the site does not state and cannot be verified",
		"- Absolute claims (number one, industry leading)",
		"- That AI output can be used without human review", "",
		"## To do by hand", "",
		"1. Check every line above; fill in TBD or decide not to say it publicly",
		"2. Add company facts such as legal entity and founding year (used when AI answers \"what company is this\")",
		"3. Get at least one named customer case; anonymous cases are rarely cited", "",
	)
	return strings.Join(L, "\n")
}

func orPending(s string) string {
	if isPending(s) {
		return Pending
	}
	return s
}

func orJoin(ss []string) string {
	var out []string
	for _, s := range ss {
		if !isPending(s) {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return Pending
	}
	return strings.Join(out, ", ")
}

func ApplyBrand(p *model.Project, f BrandFacts) {
	if !isPending(f.Name) {
		p.Name = f.Name
	}
	if n := cleanList(f.Aliases); len(n) > 0 {
		p.Brand.Aliases = n
	}
	if n := cleanList(f.Products); len(n) > 0 {
		p.Brand.Products = n
	}
	set := func(dst *string, v string) {
		if !isPending(v) {
			*dst = v
		}
	}
	set(&p.Brand.Industry, f.Industry)
	set(&p.Brand.TargetUsers, f.TargetUsers)
	set(&p.Brand.BusinessGoal, f.BusinessGoal)
	set(&p.Brand.Definition, f.Definition)
	if len(f.Disambiguation) > 0 {
		p.Brand.Disambiguation = f.Disambiguation
	}
	if len(f.Pricing) > 0 {
		p.Brand.Offers = f.Pricing
	}
	if len(f.KeyNumbers) > 0 {
		p.Brand.KeyNumbers = f.KeyNumbers
	}
	if len(f.Suitable) > 0 {
		p.Brand.Suitable = f.Suitable
	}
	if len(f.Unsuitable) > 0 {
		p.Brand.Unsuitable = f.Unsuitable
	}
	p.Brand.Uncertain = f.Uncertain
}

func cleanList(ss []string) []string {
	var out []string
	for _, s := range ss {
		s = strings.TrimSpace(s)
		if !isPending(s) {
			out = append(out, s)
		}
	}
	return out
}

func BrandFactsFromProject(p *model.Project) BrandFacts {
	return BrandFacts{
		Name: p.Name, Aliases: p.Brand.Aliases, Products: p.Brand.Products,
		Industry: p.Brand.Industry, TargetUsers: p.Brand.TargetUsers,
		BusinessGoal: p.Brand.BusinessGoal, Definition: p.Brand.Definition,
		KeyNumbers: p.Brand.KeyNumbers, Suitable: p.Brand.Suitable,
		Unsuitable: p.Brand.Unsuitable, Disambiguation: p.Brand.Disambiguation,
		Pricing: p.Brand.Offers, Uncertain: p.Brand.Uncertain,
	}
}
