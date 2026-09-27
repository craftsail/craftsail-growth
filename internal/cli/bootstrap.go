// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"fmt"

	"github.com/gotomicro/ego/task/ejob"
	"github.com/spf13/cobra"

	"github.com/craftsail/craftsail-growth/internal/invoker"
	"github.com/craftsail/craftsail-growth/internal/service/bootstrap"
	"github.com/craftsail/craftsail-growth/internal/service/sample"
)

func bootstrapCmd() *cobra.Command {
	var slug string
	var skipLLM bool
	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Derive brand facts, competitors and prompts from the site text or materials",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJob("bootstrap", func(ejob.Context) error {
				svc := bootstrap.New(invoker.DB)
				svc.LLM = sample.NewAsker()
				res, err := svc.Run(cmd.Context(), slug, skipLLM)
				if err != nil {
					return err
				}
				fmt.Printf("[craftsail-growth] bootstrap done: %d prompts, %d competitors\n", res.Questions, res.Competitors)
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "project slug")
	cmd.Flags().BoolVar(&skipLLM, "skip-llm", false, "do not call an LLM; use templates")
	_ = cmd.MarkFlagRequired("slug")
	return cmd
}
