// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/gotomicro/ego/task/ejob"
	"github.com/spf13/cobra"

	"github.com/craftsail/craftsail-growth/internal/invoker"
	"github.com/craftsail/craftsail-growth/internal/service/sample"
)

func sampleCmd() *cobra.Command {
	var slug, platforms string
	var limit, repeat int
	cmd := &cobra.Command{
		Use:   "sample",
		Short: "Ask the prompt library on each engine API",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJob("sample", func(ejob.Context) error {
				in := sample.RunInput{Limit: limit, Repeat: repeat}
				if platforms != "" {
					in.Platforms = strings.Split(platforms, ",")
				}
				res, err := sample.New(invoker.DB, sample.NewAsker()).Run(cmd.Context(), slug, in)
				if err != nil {
					return err
				}
				fmt.Printf("[craftsail-growth] sampling done: %d answers (skipped %s)\n", res.Count, strings.Join(res.Skipped, ","))
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "project slug")
	cmd.Flags().StringVar(&platforms, "platforms", "", "comma-separated engine codes")
	cmd.Flags().IntVar(&limit, "limit", 0, "max prompts per engine")
	cmd.Flags().IntVar(&repeat, "repeat", 1, "rounds per prompt")
	_ = cmd.MarkFlagRequired("slug")
	return cmd
}

func sampleSheetCmd() *cobra.Command {
	var slug, intent string
	var limit int
	cmd := &cobra.Command{
		Use:   "sample-sheet",
		Short: "Export a manual sampling sheet for engines without an API",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJob("sample-sheet", func(ejob.Context) error {
				md, err := sample.New(invoker.DB, sample.NewAsker()).Sheet(cmd.Context(), slug, intent, limit)
				if err != nil {
					return err
				}
				fmt.Print(md)
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "project slug")
	cmd.Flags().StringVar(&intent, "intent", "", "buyer = buyer-intent prompts only")
	cmd.Flags().IntVar(&limit, "limit", 0, "max prompts per engine")
	_ = cmd.MarkFlagRequired("slug")
	return cmd
}

func sampleImportCmd() *cobra.Command {
	var slug, file string
	cmd := &cobra.Command{
		Use:   "sample-import",
		Short: "Import a filled manual sampling sheet",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJob("sample-import", func(ejob.Context) error {
				b, err := os.ReadFile(file)
				if err != nil {
					return err
				}
				res, err := sample.New(invoker.DB, sample.NewAsker()).ImportMarkdown(cmd.Context(), slug, string(b))
				if err != nil {
					return err
				}
				fmt.Printf("[craftsail-growth] imported %d manual answers\n", res.Count)
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "project slug")
	cmd.Flags().StringVar(&file, "file", "", "sampling sheet markdown file")
	_ = cmd.MarkFlagRequired("slug")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}
