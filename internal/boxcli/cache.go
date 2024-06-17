// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"encoding/json"
	"fmt"
	"os/user"
	"slices"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/devbox"
	"go.jetpack.io/devbox/internal/devbox/devopt"
	"go.jetpack.io/devbox/internal/devbox/providers/identity"
	"go.jetpack.io/devbox/internal/devbox/providers/nixcache"
	nixv1alpha1 "go.jetpack.io/pkg/api/gen/priv/nix/v1alpha1"
)

type cacheFlags struct {
	pathFlag
	to string
}

type credentialsFlags struct {
	format string
}

func cacheCmd() *cobra.Command {
	flags := cacheFlags{}
	cacheCommand := &cobra.Command{
		Use:               "cache",
		Short:             "Collection of commands to interact with nix cache",
		PersistentPreRunE: ensureNixInstalled,
	}

	uploadCommand := &cobra.Command{
		Use:     "upload [installable]",
		Aliases: []string{"copy"}, // This mimics the nix command
		Short:   "upload specified or nix packages in current project to cache",
		Long: heredoc.Doc(`
			Upload specified nix installable or nix packages in current project to cache.
			If [installable] is provided, only that installable will be uploaded.
			Otherwise, all packages in the project will be uploaded.
			To upload to specific cache, use --to flag. Otherwise, a cache from
			the cache provider will be used, if available.
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return devbox.UploadInstallableToCache(
					cmd.Context(), cmd.ErrOrStderr(), flags.to, args[0],
				)
			}
			box, err := devbox.Open(&devopt.Opts{
				Dir:    flags.path,
				Stderr: cmd.ErrOrStderr(),
			})
			if err != nil {
				return errors.WithStack(err)
			}
			return box.UploadProjectToCache(cmd.Context(), flags.to)
		},
	}

	flags.pathFlag.register(uploadCommand)
	uploadCommand.Flags().StringVar(
		&flags.to, "to", "", "URI of the cache to copy to")

	cacheCommand.AddCommand(uploadCommand)
	cacheCommand.AddCommand(cacheConfigureCmd())
	cacheCommand.AddCommand(cacheCredentialsCmd())
	cacheCommand.AddCommand(cacheEnableCmd())
	cacheCommand.AddCommand(cacheInfoCmd())

	return cacheCommand
}

func cacheConfigureCmd() *cobra.Command {
	username := ""
	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Configure Nix to use the Devbox cache as a substituter",
		Long: heredoc.Doc(`
			Configure Nix to use the Devbox cache as a substituter.

			If the current Nix installation is multi-user, this command grants the Nix
			daemon access to Devbox caches by making the following changes:

			- Adds the current user to Nix's list of trusted users in the system nix.conf.
			- Adds the cache credentials to ~root/.aws/config.

			Configuration requires sudo, but only needs to happen once. The changes persist
			across Devbox accounts and organizations.

			This command is a no-op for single-user Nix installs that aren't running the
			Nix daemon.
		`),
		Hidden: true,
		Args:   cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if username == "" {
				u, _ := user.Current()
				username = u.Username
			}
			return nixcache.ConfigureReprompt(cmd.Context(), username)
		},
	}
	cmd.Flags().StringVar(&username, "user", "", "")
	return cmd
}

func cacheCredentialsCmd() *cobra.Command {
	flags := credentialsFlags{}
	cmd := &cobra.Command{
		Use:    "credentials",
		Short:  "Output S3 cache credentials",
		Hidden: true,
		Args:   cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := nixcache.CachedCredentials(cmd.Context())
			if err != nil {
				return err
			}

			if flags.format == "sh" {
				fmt.Printf("export AWS_ACCESS_KEY_ID=%q\n", creds.AccessKeyID)
				fmt.Printf("export AWS_SECRET_ACCESS_KEY=%q\n", creds.SecretAccessKey)
				fmt.Printf("export AWS_SESSION_TOKEN=%q\n", creds.SessionToken)
				return nil
			}

			out, err := json.Marshal(creds)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(out)
			return err
		},
	}
	cmd.Flags().StringVar(&flags.format, "format", "json", "Output format, either json or sh")
	return cmd
}

func cacheEnableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable",
		Short: "Enable the Devbox Nix cache for your account",
		Long: heredoc.Doc(`
			Sign up or log in to a Jetify Cloud account and configure Nix to use the
			account's Nix cache.

			For more about how Devbox configures Nix, see "devbox cache configure -h".
		`),
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			auth, err := identity.AuthClient(identity.AuthRedirectCache)
			if err != nil {
				return err
			}
			sess, _ := auth.GetSessions()
			needLogin := len(sess) == 0
			if needLogin {
				_, err = auth.LoginFlow()
				if err != nil {
					return err
				}
			}

			needConfigure := !nixcache.IsConfigured(cmd.Context())
			if needConfigure {
				u, _ := user.Current()
				err = nixcache.ConfigureReprompt(cmd.Context(), u.Username)
				if err != nil {
					return err
				}
			}

			if !needConfigure && !needLogin {
				fmt.Fprintln(cmd.OutOrStdout(), "The Devbox cache is already enabled for your account.")
			}
			return nil
		},
	}
	return cmd
}

func cacheInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Output information about the nix cache",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO(gcurtis): We can also output info about the daemon config status
			// here
			caches, err := nixcache.Caches(cmd.Context())
			if err != nil {
				return err
			}
			if len(caches) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No cache configured")
			}
			for _, cache := range caches {
				isReadOnly := !slices.Contains(
					cache.GetPermissions(),
					nixv1alpha1.Permission_PERMISSION_WRITE,
				)
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"* %s %s\n",
					cache.GetUri(),
					lo.Ternary(isReadOnly, "(read-only)", ""),
				)
			}
			return nil
		},
	}
}
