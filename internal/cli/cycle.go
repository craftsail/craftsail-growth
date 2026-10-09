// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"context"
	"fmt"

	"github.com/gotomicro/ego/task/ejob"
	"github.com/spf13/cobra"

	"github.com/craftsail/craftsail-growth/internal/invoker"
	"github.com/craftsail/craftsail-growth/internal/service/audit"
	"github.com/craftsail/craftsail-growth/internal/service/jobs"
	"github.com/craftsail/craftsail-growth/internal/service/opportunity"
	"github.com/craftsail/craftsail-growth/internal/service/pipeline"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"github.com/craftsail/craftsail-growth/internal/service/report"
	"github.com/craftsail/craftsail-growth/internal/service/sample"
)

func pipe() *pipeline.Service {
	s := pipeline.New(invoker.DB)
	s.Log = func(m string) { fmt.Println("[craftsail-growth]", m) }
	return s
}

// runPeriod starts the period through the job system so a CLI run and a
// dashboard run can never overlap.
func runPeriod(ctx context.Context, slug string, opt pipeline.Options) error {
	js := jobs.New(invoker.DB)
	jobs.Bind(js, invoker.DB)
	args := map[string]any{}
	if opt.Limit != 0 {
		args["limit"] = opt.Limit
	}
	if opt.MaxPages != 0 {
		args["max_pages"] = opt.MaxPages
	}
	if opt.NoSample {
		args["no_sample"] = true
	}
	if opt.SkipLLM {
		args["skip_llm"] = true
	}
	j, err := js.Start(ctx, slug, jobs.PeriodAction, args)
	if err != nil {
		return err
	}
	return js.Wait(ctx, j.ID)
}

func newCmd() *cobra.Command {
	var url, name, slug string
	var maxPages, limit int
	var noSample, skipLLM, force bool
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a project from a URL and run its first period",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJob("new", func(ejob.Context) error {
				_, err := pipe().NewProject(cmd.Context(), project.CreateInput{
					URL: url, Name: name, Slug: slug, MaxPages: maxPages, Force: force,
				}, pipeline.Options{MaxPages: maxPages, Limit: limit, NoSample: noSample, SkipLLM: skipLLM})
				return err
			})
		},
	}
	cmd.Flags().StringVar(&url, "url", "", "website")
	cmd.Flags().StringVar(&name, "name", "", "brand name")
	cmd.Flags().StringVar(&slug, "slug", "", "project slug")
	cmd.Flags().IntVar(&maxPages, "max-pages", 25, "crawl page limit")
	cmd.Flags().IntVar(&limit, "limit", 0, "sample only the first N prompts")
	cmd.Flags().BoolVar(&noSample, "no-sample", false, "skip sampling")
	cmd.Flags().BoolVar(&skipLLM, "skip-llm", false, "derive brand facts without an LLM")
	cmd.Flags().BoolVar(&force, "force", false, "recreate the project if it exists")
	_ = cmd.MarkFlagRequired("url")
	return cmd
}

func periodServeCmd() *cobra.Command {
	var slug string
	var maxPages, limit int
	var noSample, skipLLM bool
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run one period: crawl, audit, sample, search, verify, opportunities, report",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJob("serve", func(ejob.Context) error {
				return runPeriod(cmd.Context(), slug, pipeline.Options{MaxPages: maxPages, Limit: limit, NoSample: noSample, SkipLLM: skipLLM})
			})
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "project slug")
	cmd.Flags().IntVar(&maxPages, "max-pages", 0, "crawl page limit")
	cmd.Flags().IntVar(&limit, "limit", 0, "sample only the first N prompts")
	cmd.Flags().BoolVar(&noSample, "no-sample", false, "skip sampling")
	cmd.Flags().BoolVar(&skipLLM, "skip-llm", false, "derive brand facts without an LLM")
	_ = cmd.MarkFlagRequired("slug")
	return cmd
}

func reportCmd() *cobra.Command {
	var slug, language string
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Build the report for the current window",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJob("report", func(ejob.Context) error {
				out, err := report.New(invoker.DB).Build(cmd.Context(), slug, language)
				if err != nil {
					return err
				}
				fmt.Println("[craftsail-growth] report", out.On)
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&language, "language", "", "report language: en, zh, pt (default: project setting)")
	cmd.Flags().StringVar(&slug, "slug", "", "project slug")
	_ = cmd.MarkFlagRequired("slug")
	return cmd
}

func statusCmd() *cobra.Command {
	var slug string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show visibility, audit layers and open opportunities for a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJob("status", func(ejob.Context) error {
				ctx := cmd.Context()
				p, err := project.New(invoker.DB).Get(ctx, slug)
				if err != nil {
					return err
				}
				fmt.Printf("\n%s  %s\n", p.Name, p.Site)
				if v, err := sample.New(invoker.DB, sample.NewAsker()).Measure(ctx, slug, sample.MeasureQuery{Range: "30d"}); err == nil && v != nil {
					if v.Visibility != nil && v.VisibilityCI != nil {
						fmt.Printf("  visibility %.0f%% (95%% CI %.0f-%.0f%%, n=%d, %s)\n", *v.Visibility, v.VisibilityCI.Lo, v.VisibilityCI.Hi, v.VisibilityN, v.Access)
					} else {
						fmt.Println("  visibility not measured")
					}
				}
				if rep, _ := audit.New(invoker.DB).Latest(ctx, slug); rep != nil {
					for _, l := range rep.Layers {
						fmt.Printf("  %-10v %v\n", l["name"], l["status"])
					}
				}
				items, err := opportunity.NewDB(invoker.DB).List(ctx, slug, opportunity.ListFilter{})
				if err != nil {
					return err
				}
				fmt.Printf("  %d opportunities\n", len(items))
				for i, it := range items {
					if i >= 10 {
						break
					}
					fmt.Printf("    %s %-8s %s\n", it.Priority, it.Source, it.Title)
				}
				fmt.Println()
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "project slug")
	_ = cmd.MarkFlagRequired("slug")
	return cmd
}
