// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"fmt"

	"github.com/gotomicro/ego/task/ejob"
	"github.com/spf13/cobra"

	"github.com/craftsail/craftsail-growth/internal/invoker"
	"github.com/craftsail/craftsail-growth/internal/service/jobs"
)

func webstatsCmd() *cobra.Command {
	var slug string
	cmd := &cobra.Command{
		Use:   "webstats",
		Short: "Pull Search Console and GA4 into the local database",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJob("webstats", func(ejob.Context) error {
				js := jobs.New(invoker.DB)
				jobs.Bind(js, invoker.DB)
				j, err := js.Start(cmd.Context(), slug, "webstats", nil)
				if err != nil {
					return err
				}
				fmt.Printf("[craftsail-growth] Google sync job %d started; history continues in batches.\n", j.ID)
				if err := js.Wait(cmd.Context(), j.ID); err != nil {
					return err
				}
				fmt.Printf("[craftsail-growth] Google sync job %d completed\n", j.ID)
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "project slug")
	_ = cmd.MarkFlagRequired("slug")
	return cmd
}
