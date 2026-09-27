// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gotomicro/ego/task/ejob"
	"github.com/spf13/cobra"

	"github.com/craftsail/craftsail-growth/internal/invoker"
	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/account"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

// userCmd manages dashboard users on the server itself, for example to
// regain access when every admin password is lost.
func userCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "user", Short: "Manage dashboard users"}

	cmd.AddCommand(&cobra.Command{
		Use: "list", Short: "List users",
		RunE: func(c *cobra.Command, _ []string) error {
			return runJob("user-list", func(ejob.Context) error {
				users, err := account.New(invoker.DB).List(c.Context())
				if err != nil {
					return err
				}
				if len(users) == 0 {
					fmt.Println("[craftsail-growth] no users yet. Open the dashboard to create the first admin, or: craftsail-growth user add <name> --admin")
				}
				for _, u := range users {
					state := ""
					if u.Disabled {
						state = " (disabled)"
					}
					fmt.Printf("%-32s %-7s%s\n", u.Username, u.Role, state)
				}
				return nil
			})
		},
	})

	var admin bool
	add := &cobra.Command{
		Use: "add <username>", Short: "Add a user; reads the password from stdin (it echoes; pipe it to hide it)", Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			pass, err := readPassword()
			if err != nil {
				return err
			}
			role := model.RoleMember
			if admin {
				role = model.RoleAdmin
			}
			return runJob("user-add", func(ejob.Context) error {
				u, err := account.New(invoker.DB).CreateUser(c.Context(), args[0], pass, role)
				if err == nil {
					fmt.Printf("[craftsail-growth] added %s (%s)\n", u.Username, u.Role)
				}
				return err
			})
		},
	}
	add.Flags().BoolVar(&admin, "admin", false, "make the user an admin")
	cmd.AddCommand(add)

	cmd.AddCommand(&cobra.Command{
		Use: "passwd <username>", Short: "Set a user's password from stdin and sign them out everywhere (it echoes; pipe it: printf '%s\\n' \"$PASS\" | craftsail-growth user passwd root)", Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			pass, err := readPassword()
			if err != nil {
				return err
			}
			return runJob("user-passwd", func(ejob.Context) error {
				acc := account.New(invoker.DB)
				u, err := acc.ByUsername(c.Context(), args[0])
				if err != nil {
					return err
				}
				if err := acc.SetPassword(c.Context(), u.ID, pass); err != nil {
					return err
				}
				fmt.Printf("[craftsail-growth] password set for %s\n", u.Username)
				return nil
			})
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use: "grant <username> <project-slug> <view|edit|none>", Short: "Set a member's access to a project", Args: cobra.ExactArgs(3),
		RunE: func(c *cobra.Command, args []string) error {
			access := args[2]
			if access == "none" {
				access = ""
			}
			return runJob("user-grant", func(ejob.Context) error {
				acc := account.New(invoker.DB)
				u, err := acc.ByUsername(c.Context(), args[0])
				if err != nil {
					return err
				}
				p, err := project.New(invoker.DB).Get(c.Context(), args[1])
				if err != nil {
					return err
				}
				if u.Role == model.RoleAdmin {
					return fmt.Errorf("%s is an admin and already has edit access to every project", u.Username)
				}
				if err := acc.Grant(c.Context(), u.ID, p.ID, access); err != nil {
					return err
				}
				fmt.Printf("[craftsail-growth] %s: %s on %s\n", u.Username, args[2], p.Slug)
				return nil
			})
		},
	})
	return cmd
}

// readPassword reads one line from stdin. It echoes on a terminal; pipe it
// in to avoid that: printf '%s\n' "$PASS" | craftsail-growth user passwd root
func readPassword() (string, error) {
	fmt.Fprint(os.Stderr, "Password: ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("read password: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}
