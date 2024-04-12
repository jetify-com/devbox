// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"encoding/json"
	"os/user"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/devbox"
	"go.jetpack.io/devbox/internal/devbox/devopt"
	"go.jetpack.io/devbox/internal/devbox/providers/nixcache"
)

type cacheFlags struct {
	pathFlag
	to string
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
	cacheCommand.Hidden = true

	return cacheCommand
}

func cacheConfigureCmd() *cobra.Command {
	username := ""
	cmd := &cobra.Command{
		Use:    "configure",
		Short:  "Configure Nix to use the Devbox cache as a substituter",
		Hidden: true,
		Args:   cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if username == "" {
				u, _ := user.Current()
				username = u.Username
			}
			return nixcache.Get().Configure(cmd.Context(), username)
		},
	}
	cmd.Flags().StringVar(&username, "user", "", "")
	return cmd
}

func cacheCredentialsCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "credentials",
		Short:  "Output S3 cache credentials",
		Hidden: true,
		Args:   cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := nixcache.Get().Credentials(cmd.Context())
			if err != nil {
				return err
			}
			out, err := json.Marshal(creds)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(out)
			return err
		},
	}
}
