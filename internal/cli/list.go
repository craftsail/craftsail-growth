// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"fmt"

	"github.com/gotomicro/ego/task/ejob"
	"github.com/spf13/cobra"

	"github.com/craftsail/craftsail-growth/internal/invoker"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJob("list", func(ejob.Context) error {
				items, err := project.New(invoker.DB).List(cmd.Context())
				if err != nil {
					return err
				}
				if len(items) == 0 {
					fmt.Println("[craftsail-growth] no projects yet. Try: craftsail-growth init --url https://example.com")
					return nil
				}
				qs := &repo.Questions{DB: invoker.DB}
				reps := &repo.Reports{DB: invoker.DB}
				for _, p := range items {
					n := 0
					if rows, err := qs.List(cmd.Context(), p.ID); err == nil {
						n = len(rows)
					}
					last := "—"
					if r, err := reps.Latest(cmd.Context(), p.ID); err == nil && r != nil {
						last = r.ReportOn.Format("2006-01-02")
					}
					fmt.Printf("%-20s %-22s prompts %3d  last report %s\n", p.Slug, p.Name, n, last)
				}
				return nil
			})
		},
	}
}
