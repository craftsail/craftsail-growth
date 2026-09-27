// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"fmt"
	"strings"

	"github.com/gotomicro/ego/task/ejob"
	"github.com/spf13/cobra"

	"github.com/craftsail/craftsail-growth/internal/invoker"
	"github.com/craftsail/craftsail-growth/internal/service/generate"
	"github.com/craftsail/craftsail-growth/internal/service/verify"
)

func generateCmd() *cobra.Command {
	var slug, asset string
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate fix snippets from brand facts: llms.txt, JSON-LD, definition and FAQ blocks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJob("generate", func(ejob.Context) error {
				var which []string
				if asset != "" {
					which = strings.Split(asset, ",")
				}
				res, err := generate.New(invoker.DB).Run(cmd.Context(), slug, which)
				if err != nil {
					return err
				}
				fmt.Printf("[craftsail-growth] generated %d files; review them before publishing\n", len(res.Assets))
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "project slug")
	cmd.Flags().StringVar(&asset, "asset", "", "llms,jsonld,snippets (default all)")
	_ = cmd.MarkFlagRequired("slug")
	return cmd
}

func verifyCmd() *cobra.Command {
	var slug string
	var noRecrawl bool
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Check accepted actions against their acceptance criteria",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJob("verify", func(ejob.Context) error {
				rep, err := verify.New(invoker.DB).Run(cmd.Context(), slug, !noRecrawl)
				if err != nil {
					return err
				}
				fmt.Printf("[craftsail-growth] verify: %d pass, %d fail, %d pending; %d status changes\n",
					rep.Pass, rep.Fail, rep.Manual, rep.Changed)
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "project slug")
	cmd.Flags().BoolVar(&noRecrawl, "no-recrawl", false, "use the latest audit instead of crawling again")
	_ = cmd.MarkFlagRequired("slug")
	return cmd
}
