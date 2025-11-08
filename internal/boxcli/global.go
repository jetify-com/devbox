// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"go.jetify.com/devbox/internal/devbox"
	"go.jetify.com/devbox/internal/devbox/devopt"
	"go.jetify.com/devbox/internal/ux"
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

	addCommandAndHideConfigFlag(globalCmd, addCmd())
	addCommandAndHideConfigFlag(globalCmd, installCmd())
	addCommandAndHideConfigFlag(globalCmd, pathCmd())
	addCommandAndHideConfigFlag(globalCmd, pullCmd())
	addCommandAndHideConfigFlag(globalCmd, pushCmd())
	addCommandAndHideConfigFlag(globalCmd, removeCmd())
	addCommandAndHideConfigFlag(globalCmd, runCmd(runFlagDefaults{
		omitNixEnv: true,
	}))
	addCommandAndHideConfigFlag(globalCmd, servicesCmd(persistentPreRunE))
	addCommandAndHideConfigFlag(globalCmd, shellEnvCmd(shellenvFlagDefaults{
		omitNixEnv: true,
	}))
	addCommandAndHideConfigFlag(globalCmd, updateCmd())
	addCommandAndHideConfigFlag(globalCmd, listCmd())

	return globalCmd
}

func addCommandAndHideConfigFlag(parent, child *cobra.Command) {
	parent.AddCommand(child)
	_ = child.Flags().MarkHidden("config")
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
	err = devbox.EnsureConfig(globalConfigPath)
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
		ux.Fwarningf(
			cmd.ErrOrStderr(),
			`devbox global is not activated.

Add the following line to your shell's rcfile and restart your shell:

For bash/zsh (~/.bashrc or ~/.zshrc):
	eval "$(devbox global shellenv)"

For fish (~/.config/fish/config.fish):
	devbox global shellenv --format fish | source

For nushell (~/.config/nushell/config.nu or ~/.config/nushell/env.nu):
	devbox global shellenv --format nushell | save -f ~/.cache/devbox-env.nu
	source ~/.cache/devbox-env.nu
`,
		)
	}
	return nil
}
