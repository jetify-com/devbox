// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"go.jetpack.io/devbox/internal/devbox"
	"go.jetpack.io/devbox/internal/devbox/devopt"
	"go.jetpack.io/devbox/internal/ux"
)

func globalCmd() *cobra.Command {
	globalCmd := &cobra.Command{}
	persistentPreRunE := setGlobalConfigForDelegatedCommands(globalCmd)
	*globalCmd = cobra.Command{
		Use:   "global",
		Short: "Manage global devbox packages",
		// PersistentPreRunE is inherited only if children do not implement it
		// (i.e. it's not chained). So this is fragile. Ideally we stop
		// using PersistentPreRunE. For now a hack is to pass it down to commands
		// that declare their own.
		PersistentPreRunE:  persistentPreRunE,
		PersistentPostRunE: ensureGlobalEnvEnabled,
	}

	shellEnv := shellEnvCmd()
	// For `devbox shellenv` the default value of recompute is true.
	// Change the default value to false for `devbox global shellenv` only.
	shellEnv.Flag("recompute").DefValue = "false"
	if err := shellEnv.Flag("recompute").Value.Set("false"); err != nil {
		panic(errors.WithStack(err))
	}

	addCommandAndHideConfigFlag(globalCmd, addCmd())
	addCommandAndHideConfigFlag(globalCmd, installCmd())
	addCommandAndHideConfigFlag(globalCmd, pathCmd())
	addCommandAndHideConfigFlag(globalCmd, pullCmd())
	addCommandAndHideConfigFlag(globalCmd, pushCmd())
	addCommandAndHideConfigFlag(globalCmd, removeCmd())
	addCommandAndHideConfigFlag(globalCmd, runCmd())
	addCommandAndHideConfigFlag(globalCmd, servicesCmd(persistentPreRunE))
	addCommandAndHideConfigFlag(globalCmd, shellEnv)
	addCommandAndHideConfigFlag(globalCmd, updateCmd())
	addCommandAndHideConfigFlag(globalCmd, listCmd())

	return globalCmd
}

func addCommandAndHideConfigFlag(parent, child *cobra.Command) {
	parent.AddCommand(child)
	_ = child.Flags().MarkHidden("config")
}

type listCmdFlags struct {
	config configFlags
}

func listCmd() *cobra.Command {
	flags := listCmdFlags{}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List global packages",
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			box, err := devbox.Open(&devopt.Opts{
				Dir:    flags.config.path,
				Stderr: cmd.ErrOrStderr(),
			})
			if err != nil {
				return errors.WithStack(err)
			}
			for _, p := range box.AllPackageNamesIncludingRemovedTriggerPackages() {
				fmt.Fprintf(cmd.OutOrStdout(), "* %s\n", p)
			}
			return nil
		},
	}
	flags.config.register(cmd)
	return cmd
}

var globalConfigPath string

func ensureGlobalConfig() (string, error) {
	if globalConfigPath != "" {
		return globalConfigPath, nil
	}

	globalConfigPath, err := devbox.GlobalDataPath()
	if err != nil {
		return "", err
	}
	_, err = devbox.InitConfig(globalConfigPath)
	if err != nil {
		return "", err
	}
	return globalConfigPath, nil
}

func setGlobalConfigForDelegatedCommands(
	globalCmd *cobra.Command,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		globalPath, err := ensureGlobalConfig()
		if err != nil {
			return err
		}

		for _, c := range globalCmd.Commands() {
			if f := c.Flag("config"); f != nil && f.Value.Type() == "string" {
				if err := f.Value.Set(globalPath); err != nil {
					return errors.WithStack(err)
				}
			}
		}
		return nil
	}
}

func ensureGlobalEnvEnabled(cmd *cobra.Command, args []string) error {
	if cmd.Name() == "shellenv" {
		return nil
	}
	path, err := ensureGlobalConfig()
	if err != nil {
		return errors.WithStack(err)
	}

	box, err := devbox.Open(&devopt.Opts{
		Dir:    path,
		Stderr: cmd.ErrOrStderr(),
	})
	if err != nil {
		return err
	}
	if !box.IsEnvEnabled() {
		fmt.Fprintln(cmd.ErrOrStderr())
		ux.Fwarning(
			cmd.ErrOrStderr(),
			`devbox global is not activated.

Add the following line to your shell's rcfile (e.g., ~/.bashrc or ~/.zshrc)
and restart your shell to fix this:

	eval "$(devbox global shellenv)"
`,
		)
	}
	return nil
}
