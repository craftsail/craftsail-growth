// SPDX-License-Identifier: AGPL-3.0-or-later

package jobs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/pkg/httputil"
	"github.com/craftsail/craftsail-growth/internal/service/audit"
	"github.com/craftsail/craftsail-growth/internal/service/bootstrap"
	"github.com/craftsail/craftsail-growth/internal/service/crawl"
	"github.com/craftsail/craftsail-growth/internal/service/generate"
	"github.com/craftsail/craftsail-growth/internal/service/opportunity"
	"github.com/craftsail/craftsail-growth/internal/service/pipeline"
	"github.com/craftsail/craftsail-growth/internal/service/report"
	"github.com/craftsail/craftsail-growth/internal/service/sample"
	"github.com/craftsail/craftsail-growth/internal/service/verify"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

// Bind registers every background action. The dashboard, the scheduler and
// the CLI all start work through these specs.
func Bind(s *Service, db *gorm.DB) {
	s.RegisterSpec("first-check", Spec{Label: "Check website", Desc: "Crawl up to five pages and save a technical audit; no model key required",
		Run: func(ctx context.Context, slug string, args map[string]any, log func(string)) error {
			pipe := pipeline.New(db)
			pipe.Log = log
			return pipe.FirstCheck(ctx, slug, intArg(args, "max_pages", "max-pages"))
		}})
	s.RegisterSpec("crawl", Spec{Label: "Crawl site", Desc: "Fetch the site's pages again", Args: []string{"--max-pages"},
		Run: func(ctx context.Context, slug string, args map[string]any, log func(string)) error {
			res, err := crawl.New(db, httputil.New()).Run(ctx, slug, intArg(args, "max_pages", "max-pages"))
			if res != nil {
				log(fmt.Sprintf("%d of %d pages reachable", res.PagesOK, res.PagesCrawled))
			}
			return err
		}})
	s.RegisterSpec("audit", Spec{Label: "Audit site", Desc: "Run the SEO and GEO rules on the crawled pages",
		Run: func(ctx context.Context, slug string, args map[string]any, log func(string)) error {
			rep, err := audit.New(db).Run(ctx, slug)
			if rep != nil {
				log(fmt.Sprintf("%d pages, %d findings", rep.PageCount, len(rep.Findings)))
			}
			return err
		}})
	s.RegisterSpec("bootstrap", Spec{Label: "Derive brand facts", Desc: "Brand facts, competitors and prompts from the site text", Slow: true, Args: []string{"--skip-llm"},
		Run: func(ctx context.Context, slug string, args map[string]any, log func(string)) error {
			svc := bootstrap.New(db)
			svc.LLM = sample.NewAsker()
			res, err := svc.Run(ctx, slug, boolArg(args, "skip_llm", "skip-llm"))
			if res != nil && res.SkipLLM && !boolArg(args, "skip_llm", "skip-llm") {
				log("no model answered; wrote template prompts instead")
			}
			if res != nil {
				log(fmt.Sprintf("%d prompts, %d competitors", res.Questions, res.Competitors))
			}
			return err
		}})
	s.RegisterSpec("sample", Spec{Label: "Sample AI answers", Desc: "Ask the prompt library on each engine", Slow: true, Args: []string{"--limit", "--repeat", "--platforms"},
		Run: func(ctx context.Context, slug string, args map[string]any, log func(string)) error {
			in := sample.RunInput{Limit: intArg(args, "limit"), Repeat: intArg(args, "repeat"), RetryRun: uint64(intArg(args, "retry_run"))}
			if p := strArg(args, "platforms"); p != "" {
				in.Platforms = strings.Split(p, ",")
			}
			res, err := sample.New(db, sample.NewAsker()).Run(ctx, slug, in)
			if res != nil {
				log(fmt.Sprintf("%d answers, skipped %s", res.Count, strings.Join(res.Skipped, ",")))
			}
			return err
		}})
	s.RegisterSpec("indexing", Spec{Resumable: true, Label: "Discover URLs and inspect indexing", Desc: "Read sitemaps and rotate URL Inspection independently of traffic history", Slow: true,
		Run: func(ctx context.Context, slug string, args map[string]any, log func(string)) error {
			out, err := webstats.New(db).RunIndexBatch(ctx, slug, time.Unix(int64(intArg(args, "_started_at")), 0))
			if out != nil {
				if out.Note != "" {
					log(out.Note)
				}
				if !out.Connected {
					log("URL inventory updated; connect Google to inspect indexing")
				}
				if err == nil && out.Pending {
					return &Deferred{After: max(time.Second, time.Until(time.Unix(out.RetryAt, 0)))}
				}
			}
			return err
		}})
	s.RegisterSpec("webstats", Spec{Resumable: true, Label: "Sync Google", Desc: "Pull Search Console and GA4 into the local database", Slow: true,
		Run: func(ctx context.Context, slug string, args map[string]any, log func(string)) error {
			res, err := webstats.New(db).RunBatch(ctx, slug, webstats.BatchOptions{RefreshBefore: time.Unix(int64(intArg(args, "_started_at")), 0)})
			if res != nil {
				log(fmt.Sprintf("%d query rows, %d GA4 rows; batch used %d requests and %d date partitions", res.GscRows, res.GaRows, res.Requests, res.Batches))
				if res.IndexNote != "" {
					log(res.IndexNote)
				}
				if err == nil && res.Pending {
					after := 15 * time.Second
					if res.RetryAt > 0 {
						after = max(time.Second, time.Until(time.Unix(res.RetryAt, 0)))
					}
					return &Deferred{After: after}
				}
			}
			return err
		}})
	s.RegisterSpec("opportunities", Spec{Label: "Collect opportunities", Desc: "Audit, AI citation, search and metric sources in one ranked list",
		Run: func(ctx context.Context, slug string, args map[string]any, log func(string)) error {
			items, err := opportunity.NewDB(db).List(ctx, slug, opportunity.ListFilter{})
			if err != nil {
				return err
			}
			p0 := 0
			for _, it := range items {
				if it.Priority == "P0" {
					p0++
				}
			}
			log(fmt.Sprintf("%d opportunities, %d at P0", len(items), p0))
			return nil
		}})
	s.RegisterSpec("generate", Spec{Label: "Generate fix snippets", Desc: "llms.txt, JSON-LD, definition and FAQ blocks from brand facts", Args: []string{"--asset"},
		Run: func(ctx context.Context, slug string, args map[string]any, log func(string)) error {
			var which []string
			if a := strArg(args, "asset"); a != "" {
				which = strings.Split(a, ",")
			}
			res, err := generate.New(db).Run(ctx, slug, which)
			if res != nil {
				log(fmt.Sprintf("%d files", len(res.Assets)))
			}
			return err
		}})
	s.RegisterSpec("report", Spec{Label: "Build report", Desc: "Markdown and HTML",
		Run: func(ctx context.Context, slug string, args map[string]any, log func(string)) error {
			language, _ := args["language"].(string)
			out, err := report.New(db).Build(ctx, slug, language)
			if out != nil {
				log("report " + out.On)
			}
			return err
		}})
	s.RegisterSpec("verify", Spec{Label: "Verify actions", Desc: "Crawl again and check accepted actions", Slow: true, Args: []string{"--no-recrawl"},
		Run: func(ctx context.Context, slug string, args map[string]any, log func(string)) error {
			rep, err := verify.New(db).Run(ctx, slug, !boolArg(args, "no_recrawl", "no-recrawl"))
			if rep != nil {
				log(fmt.Sprintf("%d pass, %d fail, %d pending", rep.Pass, rep.Fail, rep.Manual))
			}
			return err
		}})
	s.RegisterSpec("sample-sheet", Spec{Label: "Export manual sampling sheet", Desc: "For engines without an API",
		Run: func(ctx context.Context, slug string, args map[string]any, log func(string)) error {
			md, err := sample.New(db, sample.NewAsker()).Sheet(ctx, slug, strArg(args, "intent"), intArg(args, "limit"))
			if err != nil {
				return err
			}
			log(fmt.Sprintf("sheet of %d characters", len(md)))
			return nil
		}})
	period := func(ctx context.Context, slug string, args map[string]any, log func(string)) error {
		pipe := pipeline.New(db)
		pipe.Log = log
		return pipe.Period(ctx, slug, pipeline.Options{
			MaxPages: intArg(args, "max_pages", "max-pages"), Limit: intArg(args, "limit"),
			NoSample: boolArg(args, "no_sample", "no-sample"), SkipLLM: boolArg(args, "skip_llm", "skip-llm"),
			Scheduled: boolArg(args, "monitor"),
		})
	}
	s.RegisterSpec(PeriodAction, Spec{Label: PeriodLabel, Desc: "crawl, audit, sample, search, verify, opportunities, report", Slow: true,
		Args: []string{"--max-pages", "--limit", "--no-sample", "--skip-llm"}, Run: period})
}

func intArg(args map[string]any, keys ...string) int {
	for _, k := range keys {
		if v, ok := args[k]; ok {
			switch t := v.(type) {
			case int:
				return t
			case int64:
				return int(t)
			case float64:
				return int(t)
			case string:
				n := 0
				for _, r := range t {
					if r < '0' || r > '9' {
						continue
					}
					n = n*10 + int(r-'0')
				}
				return n
			}
		}
	}
	return 0
}

func boolArg(args map[string]any, keys ...string) bool {
	for _, k := range keys {
		if v, ok := args[k]; ok {
			switch t := v.(type) {
			case bool:
				return t
			case string:
				return t == "1" || t == "true" || t == "yes"
			}
		}
	}
	return false
}

func strArg(args map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := args[k]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}
