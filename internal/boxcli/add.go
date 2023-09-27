// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/impl/devopt"
	"go.jetpack.io/devbox/internal/nix"
)

const toSearchForPackages = "To search for packages, use the `devbox search` command"

type addCmdFlags struct {
	config           configFlags
	allowInsecure    bool
	platforms        []string
	excludePlatforms []string
}

func addCmd() *cobra.Command {
	flags := addCmdFlags{}

	command := &cobra.Command{
		Use:     "add <pkg>...",
		Short:   "Add a new package to your devbox",
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				fmt.Fprintf(
					cmd.ErrOrStderr(),
					"Usage: %s\n\n%s\n",
					cmd.UseLine(),
					toSearchForPackages,
				)
				return nil
			}
			err := addCmdFunc(cmd, args, flags)
			if errors.Is(err, nix.ErrPackageNotFound) {
				return usererr.WithUserMessage(err, toSearchForPackages)
			}
			return err
		},
	}

	flags.config.register(command)
	command.Flags().BoolVar(
		&flags.allowInsecure, "allow-insecure", false,
		"allow adding packages marked as insecure.")
	command.Flags().StringSliceVarP(
		&flags.platforms, "platform", "p", []string{},
		"add packages to run on only this platform.")
	command.Flags().StringSliceVarP(
		&flags.excludePlatforms, "exclude-platform", "e", []string{},
		"exclude packages from a specific platform.")

	return command
}

func addCmdFunc(cmd *cobra.Command, args []string, flags addCmdFlags) error {
	box, err := devbox.Open(&devopt.Opts{
		Dir:               flags.config.path,
		Stderr:            cmd.ErrOrStderr(),
		AllowInsecureAdds: flags.allowInsecure,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	return box.Add(cmd.Context(), flags.platforms, flags.excludePlatforms, args...)
}
