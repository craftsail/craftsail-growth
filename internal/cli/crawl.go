// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"fmt"

	"github.com/gotomicro/ego/task/ejob"
	"github.com/spf13/cobra"

	"github.com/craftsail/craftsail-growth/internal/invoker"
	"github.com/craftsail/craftsail-growth/internal/pkg/httputil"
	"github.com/craftsail/craftsail-growth/internal/service/audit"
	"github.com/craftsail/craftsail-growth/internal/service/crawl"
)

func crawlCmd() *cobra.Command {
	var slug string
	var maxPages int
	cmd := &cobra.Command{
		Use:   "crawl",
		Short: "Crawl the site",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJob("crawl", func(ejob.Context) error {
				svc := crawl.New(invoker.DB, httputil.New())
				res, err := svc.Run(cmd.Context(), slug, maxPages)
				if res != nil {
					fmt.Printf("[craftsail-growth] crawl done: %d of %d pages reachable\n", res.PagesOK, res.PagesCrawled)
				}
				return err
			})
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "project slug")
	cmd.Flags().IntVar(&maxPages, "max-pages", 0, "page limit")
	_ = cmd.MarkFlagRequired("slug")
	return cmd
}

func auditCmd() *cobra.Command {
	var slug string
	var export bool
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Audit the crawled pages (SEO and GEO rules)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJob("audit", func(ejob.Context) error {
				svc := audit.New(invoker.DB)
				if export {
					md, err := svc.ExportMarkdown(cmd.Context(), slug)
					if err != nil {
						return err
					}
					fmt.Print(md)
					return nil
				}
				rep, err := svc.Run(cmd.Context(), slug)
				if err != nil {
					return err
				}
				fmt.Printf("[craftsail-growth] audit done: %d pages, %d findings\n", rep.PageCount, len(rep.Findings))
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "project slug")
	cmd.Flags().BoolVar(&export, "export", false, "print the audit as Markdown for an AI assistant")
	_ = cmd.MarkFlagRequired("slug")
	return cmd
}
