// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"fmt"
	"os"

	"github.com/gotomicro/ego/task/ejob"
	"github.com/spf13/cobra"

	"github.com/craftsail/craftsail-growth/internal/invoker"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

func initCmd() *cobra.Command {
	var in project.CreateInput
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a project without running a period",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJob("init", func(ejob.Context) error {
				if in.Materials != "" {
					if b, err := os.ReadFile(in.Materials); err == nil {
						in.Materials = string(b)
					}
				}
				p, err := project.New(invoker.DB).Create(cmd.Context(), in)
				if err != nil {
					return err
				}
				fmt.Printf("[craftsail-growth] project created: %s (brand %s)\n", p.Slug, p.Name)
				if p.NoSite {
					fmt.Println("[craftsail-growth] no-site project: crawl, audit and site snippets do not apply")
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&in.URL, "url", "", "website URL; leave empty and pass --no-site if there is none")
	cmd.Flags().BoolVar(&in.NoSite, "no-site", false, "the brand has no website")
	cmd.Flags().StringVar(&in.Materials, "materials", "", "brand materials: a file path or the text itself")
	cmd.Flags().StringVar(&in.Name, "name", "", "brand name")
	cmd.Flags().StringVar(&in.Slug, "slug", "", "project slug")
	cmd.Flags().IntVar(&in.MaxPages, "max-pages", 25, "crawl page limit")
	cmd.Flags().BoolVar(&in.Force, "force", false, "recreate the project if it exists")
	return cmd
}
