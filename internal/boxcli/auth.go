package boxcli

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/auth"
)

func AuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "auth",
	}

	cmd.AddCommand(logoutCmd())
	cmd.AddCommand(whoamiCmd())
	return cmd
}

func logoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Logs out the user with devbox",
		RunE: func(cmd *cobra.Command, args []string) error {
			return auth.Clear()
		},
	}

	return cmd
}

func whoamiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Prints information about the current user",
		RunE: func(cmd *cobra.Command, args []string) error {
			source, username, err := auth.Username()
			if err != nil {
				return errors.WithStack(err)
			}
			w := cmd.OutOrStdout()
			if source == auth.NoneSource {
				fmt.Fprintf(w, "Not logged in\n")
			} else {
				fmt.Fprintf(w, "Username: %s\nSource: %s\n", username, source)
			}
			return nil
		},
	}
	return cmd
}
