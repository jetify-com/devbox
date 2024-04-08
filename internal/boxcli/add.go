// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/devbox"
	"go.jetpack.io/devbox/internal/devbox/devopt"
	"go.jetpack.io/devbox/internal/nix"
)

const toSearchForPackages = "To search for packages, use the `devbox search` command"

type addCmdFlags struct {
	config           configFlags
	allowInsecure    []string
	disablePlugin    bool
	platforms        []string
	excludePlatforms []string
	patchGlibc       bool
	outputs          []string
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
	command.Flags().StringSliceVar(
		&flags.allowInsecure, "allow-insecure", []string{},
		"allow adding packages marked as insecure.")
	command.Flags().BoolVar(
		&flags.disablePlugin, "disable-plugin", false,
		"disable plugin (if any) for this package.")
	command.Flags().StringSliceVarP(
		&flags.platforms, "platform", "p", []string{},
		"add packages to run on only this platform.")
	command.Flags().StringSliceVarP(
		&flags.excludePlatforms, "exclude-platform", "e", []string{},
		"exclude packages from a specific platform.")
	command.Flags().BoolVar(
		&flags.patchGlibc, "patch-glibc", false,
		"patch any ELF binaries to use the latest glibc version in nixpkgs")
	command.Flags().StringSliceVarP(
		&flags.outputs, "outputs", "o", []string{},
		"specify the outputs to select for the nix package")

	return command
}

func addCmdFunc(cmd *cobra.Command, args []string, flags addCmdFlags) error {
	box, err := devbox.Open(&devopt.Opts{
		Dir:         flags.config.path,
		Environment: flags.config.environment,
		Stderr:      cmd.ErrOrStderr(),
	})
	if err != nil {
		return errors.WithStack(err)
	}

	return box.Add(cmd.Context(), args, devopt.AddOpts{
		AllowInsecure:    flags.allowInsecure,
		DisablePlugin:    flags.disablePlugin,
		Platforms:        flags.platforms,
		ExcludePlatforms: flags.excludePlatforms,
		PatchGlibc:       flags.patchGlibc,
		Outputs:          flags.outputs,
	})
}
