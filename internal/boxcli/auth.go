// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/auth"
)

func authCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "auth",
		Short:  "Devbox auth commands",
		Hidden: true,
	}

	cmd.AddCommand(loginCmd())
	cmd.AddCommand(refreshCmd())

	return cmd
}

func loginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "login",
		Short:  "Login to devbox",
		Args:   cobra.ExactArgs(0),
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return auth.NewAuthenticator().DeviceAuthFlow(cmd.Context())
		},
	}

	return cmd
}

func refreshCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "refresh",
		Short:  "Refresh credentials",
		Args:   cobra.ExactArgs(0),
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return auth.NewAuthenticator().RefreshTokens()
		},
	}

	return cmd
}
