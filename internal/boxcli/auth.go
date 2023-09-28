// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/envir"
	"go.jetpack.io/pkg/sandbox/auth"
	"go.jetpack.io/pkg/sandbox/auth/session"
)

var (
	issuer   = envir.GetValueOrDefault("DEVBOX_AUTH_ISSUER", "https://accounts.jetpack.io")
	clientID = envir.GetValueOrDefault("DEVBOX_AUTH_CLIENT_ID", "ff3d4c9c-1ac8-42d9-bef1-f5218bb1a9f6")
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
			c, err := auth.NewClient(issuer, clientID)
			if err != nil {
				return err
			}
			t, err := c.LoginFlow()
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Logged in as : %s\n", t.IDClaims().Email)
			return nil
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
			c, err := auth.NewClient(issuer, clientID)
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

// This is for debugging purposes only. Hidden.
func refreshCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "refresh",
		Short:  "Refresh credentials",
		Args:   cobra.ExactArgs(0),
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := auth.NewClient(issuer, clientID)
			if err != nil {
				return err
			}
			_ = c.RefreshSession()
			fmt.Fprintln(cmd.OutOrStdout(), "Refreshed successfully")
			return nil
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
			tok, err := genSession()
			if err != nil {
				return err
			} else if tok == nil {
				return usererr.New("not logged in")
			}
			idClaims := tok.IDClaims()

			fmt.Fprintf(cmd.OutOrStdout(), "Logged in\n")
			fmt.Fprintf(cmd.OutOrStdout(), "User ID: %s\n", idClaims.Subject)

			if idClaims.OrgID != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Org ID: %s\n", idClaims.OrgID)
			}

			if idClaims.Email != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Email: %s\n", idClaims.Email)
			}

			if idClaims.Name != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Name: %s\n", idClaims.Name)
			}

			return nil
		},
	}

	return cmd
}

func genSession() (*session.Token, error) {
	c, err := auth.NewClient(issuer, clientID)
	if err != nil {
		return nil, err
	}
	return c.GetSession(), nil
}
