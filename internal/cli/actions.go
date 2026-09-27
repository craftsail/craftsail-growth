// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/gotomicro/ego/task/ejob"
	"github.com/spf13/cobra"

	"github.com/craftsail/craftsail-growth/internal/invoker"
	"github.com/craftsail/craftsail-growth/internal/service/opportunity"
	"github.com/craftsail/craftsail-growth/internal/service/plan"
)

func opportunitiesCmd() *cobra.Command {
	var slug, accept, dismiss, source, status string
	cmd := &cobra.Command{
		Use:   "opportunities",
		Short: "List opportunities, or accept / dismiss one by key",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJob("opportunities", func(ejob.Context) error {
				svc := opportunity.NewDB(invoker.DB)
				ctx := cmd.Context()
				switch {
				case accept != "":
					t, err := svc.Accept(ctx, slug, accept)
					if err != nil {
						return err
					}
					fmt.Printf("[craftsail-growth] accepted as %s\n", t.Code)
					return nil
				case dismiss != "":
					if _, err := svc.Dismiss(ctx, slug, dismiss); err != nil {
						return err
					}
					fmt.Println("[craftsail-growth] dismissed")
					return nil
				}
				items, err := svc.List(ctx, slug, opportunity.ListFilter{Source: source, Status: status})
				if err != nil {
					return err
				}
				for _, it := range items {
					st := it.Status
					if st == "" {
						st = "new"
					}
					fmt.Printf("%s  %-8s  %-10s  %s\n    key: %s\n", it.Priority, it.Source, st, it.Title, it.Key)
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "project slug")
	cmd.Flags().StringVar(&accept, "accept", "", "accept the opportunity with this key")
	cmd.Flags().StringVar(&dismiss, "dismiss", "", "dismiss the opportunity with this key")
	cmd.Flags().StringVar(&source, "source", "", "audit | citation | search | metric")
	cmd.Flags().StringVar(&status, "status", "", "new | open | doing | done | verified | regressed | dismissed")
	_ = cmd.MarkFlagRequired("slug")
	return cmd
}

func taskCmd() *cobra.Command {
	var slug, id, status, note string
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Show an action, or set its status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJob("task", func(ejob.Context) error {
				svc := plan.New(invoker.DB)
				if status == "" {
					t, err := svc.Get(cmd.Context(), slug, id)
					if err != nil {
						return err
					}
					b, _ := json.MarshalIndent(t, "", "  ")
					fmt.Println(string(b))
					return nil
				}
				t, err := svc.SetStatus(cmd.Context(), slug, id, status, note)
				if err != nil {
					return err
				}
				fmt.Printf("[craftsail-growth] %s -> %s\n", t.Code, t.Status)
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "project slug")
	cmd.Flags().StringVar(&id, "id", "", "action code, e.g. A-001")
	cmd.Flags().StringVar(&status, "status", "", "open | doing | done | dismissed")
	cmd.Flags().StringVar(&note, "note", "", "note")
	_ = cmd.MarkFlagRequired("slug")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}
