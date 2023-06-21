// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/auth"
)

func authCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Devbox auth commands",
	}

	cmd.AddCommand(loginCmd())
	cmd.AddCommand(logoutCmd())
	cmd.AddCommand(refreshCmd())
	cmd.AddCommand(whoAmICmd())

	return cmd
}

func loginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to devbox",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return auth.NewAuthenticator().DeviceAuthFlow(
				cmd.Context(),
				cmd.OutOrStdout(),
			)
		},
	}

	return cmd
}

func logoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "logout from devbox",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := auth.NewAuthenticator().Logout()
			if err == nil {
				fmt.Fprintln(cmd.OutOrStdout(), "Logged out successfully")
			}
			return err
		},
	}

	return cmd
}

// This is for debugging purposes only. Hidden.
func refreshCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "refresh",
		Short:  "Refresh credentials",
		Args:   cobra.ExactArgs(0),
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := auth.NewAuthenticator().RefreshTokens()
			return err
		},
	}

	return cmd
}

func whoAmICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Show the current user",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			user, err := auth.GetUser()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), user)
			return nil
		},
	}

	return cmd
}
