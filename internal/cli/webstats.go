// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"fmt"

	"github.com/gotomicro/ego/task/ejob"
	"github.com/spf13/cobra"

	"github.com/craftsail/craftsail-growth/internal/invoker"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

func webstatsCmd() *cobra.Command {
	var slug string
	cmd := &cobra.Command{
		Use:   "webstats",
		Short: "Pull Search Console and GA4 into the local database",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJob("webstats", func(ejob.Context) error {
				res, err := webstats.New(invoker.DB).Run(cmd.Context(), slug)
				if err != nil {
					return err
				}
				fmt.Printf("[craftsail-growth] Google sync: %d query rows, %d GA4 rows, %s to %s\n", res.GscRows, res.GaRows, res.From, res.To)
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "project slug")
	_ = cmd.MarkFlagRequired("slug")
	return cmd
}
