// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/craftsail/craftsail-growth/internal/pkg/dotenv"
)

var cfgFile string

func Execute() {
	_ = dotenv.Load(dotenv.FindRoot())
	root := &cobra.Command{
		Use:           "craftsail-growth",
		Short:         "craftsail-growth: measure and improve brand visibility in AI answers and search",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Parent() != nil && cmd.Parent().Name() == "docs" {
				return nil // docs generation needs no config
			}
			created, err := ensureConfig(cfgFile)
			if err != nil {
				return err
			}
			if created {
				_, _ = os.Stderr.WriteString("[craftsail-growth] created " + cfgFile + " from the example; edit dsn and [auth] before a public deploy\n")
			}
			return nil
		},
	}
	root.PersistentFlags().StringVar(&cfgFile, "config", "config/default.toml", "ego config file")
	root.AddCommand(
		serverCmd(), uiCmd(), periodServeCmd(),
		listCmd(), initCmd(), newCmd(),
		crawlCmd(), auditCmd(), bootstrapCmd(),
		sampleCmd(), webstatsCmd(), sampleSheetCmd(), sampleImportCmd(),
		opportunitiesCmd(), taskCmd(), generateCmd(),
		verifyCmd(), reportCmd(), statusCmd(), userCmd(),
	)
	if err := root.Execute(); err != nil {
		_, _ = os.Stderr.WriteString("[craftsail-growth] error: " + err.Error() + "\n")
		os.Exit(1)
	}
}

func egoArgs(extra ...string) []string {
	args := []string{"--config", cfgFile}
	return append(args, extra...)
}
