// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/devbox/internal/integrations/envsec"
	"go.jetpack.io/pkg/auth"
	"go.jetpack.io/pkg/auth/session"
)

// This matches default scopes for envsec. TODO: export this in envsec.
var scopes = []string{"openid", "offline_access", "email", "profile"}

func authCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Devbox auth commands",
	}

	cmd.AddCommand(loginCmd())
	cmd.AddCommand(logoutCmd())
	cmd.AddCommand(whoAmICmd())

	return cmd
}

func loginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to devbox",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := auth.NewClient(build.Issuer(), build.ClientID(), scopes)
			if err != nil {
				return err
			}
			t, err := c.LoginFlow()
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Logged in as: %s\n", t.IDClaims().Email)
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
			c, err := auth.NewClient(build.Issuer(), build.ClientID(), scopes)
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

func whoAmICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Show the current user",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			return envsec.DefaultEnvsec(cmd.ErrOrStderr(), wd).
				WhoAmI(cmd.Context(), cmd.OutOrStdout(), false)
		},
	}

	return cmd
}

func genSession(ctx context.Context) (*session.Token, error) {
	c, err := auth.NewClient(build.Issuer(), build.ClientID(), scopes)
	if err != nil {
		return nil, err
	}
	return c.GetSession(ctx)
}
