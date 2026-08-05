// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"go.jetify.com/devbox/internal/build"
	"go.jetify.com/devbox/internal/devbox/providers/identity"
	"go.jetify.com/devbox/internal/ux"
	"go.jetify.com/pkg/api"
)

func authCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Devbox auth commands",
	}

	cmd.AddCommand(loginCmd())
	cmd.AddCommand(logoutCmd())
	cmd.AddCommand(authNewTokenCommand())

	return cmd
}

func loginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to devbox",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := identity.AuthClient(identity.AuthRedirectDefault)
			if err != nil {
				return err
			}
			t, err := c.LoginFlow()
			if err != nil {
				return err
			}
			// TODO: all uses of IDClaims() are broken when using a static
			// non-expiring token (i.e. API_TOKEN)
			fmt.Fprintf(cmd.ErrOrStderr(), "Logged in as: %s\n", t.IDClaims().Email)
			return nil
		},
	}

	return cmd
}

func logoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout from devbox",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := identity.AuthClient(identity.AuthRedirectDefault)
			if err != nil {
				return err
			}
			err = c.LogoutFlow()
			if err == nil {
				fmt.Fprintln(cmd.OutOrStdout(), "Logged out successfully")
			}
			return err
		},
	}

	return cmd
}

func authNewTokenCommand() *cobra.Command {
	tokensCmd := &cobra.Command{
		Use:   "tokens",
		Short: "Manage devbox auth tokens",
	}

	newCmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new token",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			token, err := identity.GenSession(ctx)
			if err != nil {
				return err
			}
			client := api.NewClient(ctx, build.JetpackAPIHost(), token)
			pat, err := client.CreateToken(ctx)
			if err != nil {
				// This is a hack because errors are not returning with correct code.
				// Once that is fixed, we can switch to use *connect.Error Code() instead.
				if strings.Contains(err.Error(), "permission_denied") {
					ux.Ferrorf(
						cmd.ErrOrStderr(),
						"You do not have permission to create a token. Please contact your"+
							" administrator.",
					)
					return nil
				}
				return err
			}
			ux.Fsuccessf(cmd.OutOrStdout(), "Token created.\n\n")
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			// Row lines are configured through the renderer in the new API
			if err := table.Bulk([][]string{
				{"Token ID", pat.GetToken().GetId()},
				{"Secret", pat.GetToken().GetSecret()},
			}); err != nil {
				return err
			}
			return table.Render()
		},
	}

	tokensCmd.AddCommand(newCmd)

	return tokensCmd
}
