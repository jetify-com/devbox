// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"io/fs"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/ux"
)

type globalPullCmdFlags struct {
	force bool
}

func globalCmd() *cobra.Command {

	globalCmd := &cobra.Command{}

	*globalCmd = cobra.Command{
		Use:                "global",
		Short:              "Manage global devbox packages",
		PersistentPreRunE:  setGlobalConfigForDelegatedCommands(globalCmd),
		PersistentPostRunE: ensureGlobalEnvEnabled,
	}

	addCommandAndHideConfigFlag(globalCmd, addCmd())
	addCommandAndHideConfigFlag(globalCmd, removeCmd())
	addCommandAndHideConfigFlag(globalCmd, installCmd())
	addCommandAndHideConfigFlag(globalCmd, shellEnvCmd())

	// Create list for non-global? Mike: I want it :)
	globalCmd.AddCommand(globalListCmd())
	globalCmd.AddCommand(globalPullCmd())

	return globalCmd
}

func addCommandAndHideConfigFlag(parent *cobra.Command, child *cobra.Command) {
	parent.AddCommand(child)
	_ = child.Flags().MarkHidden("config")
}

func globalListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List global packages",
		PreRunE: ensureNixInstalled,
		RunE:    listGlobalCmdFunc,
	}
}

func globalPullCmd() *cobra.Command {
	flags := globalPullCmdFlags{}
	cmd := &cobra.Command{
		Use:     "pull <file> | <url>",
		Short:   "Pull a global config from a file or URL",
		Long:    "Pull a global config from a file or URL. URLs must be prefixed with 'http://' or 'https://'.",
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			return pullGlobalCmdFunc(cmd, args, flags.force)
		},
		Args: cobra.ExactArgs(1),
	}

	cmd.Flags().BoolVarP(
		&flags.force, "force", "f", false,
		"Force overwrite of existing global config files",
	)

	return cmd
}

func listGlobalCmdFunc(cmd *cobra.Command, args []string) error {
	path, err := ensureGlobalConfig(cmd)
	if err != nil {
		return errors.WithStack(err)
	}

	box, err := devbox.Open(path, cmd.OutOrStdout())
	if err != nil {
		return errors.WithStack(err)
	}
	return box.PrintGlobalList()
}

func pullGlobalCmdFunc(
	cmd *cobra.Command,
	args []string,
	overwrite bool,
) error {
	path, err := ensureGlobalConfig(cmd)
	if err != nil {
		return errors.WithStack(err)
	}

	box, err := devbox.Open(path, cmd.ErrOrStderr())
	if err != nil {
		return errors.WithStack(err)
	}
	err = box.PullGlobal(cmd.Context(), overwrite, args[0])
	if errors.Is(err, fs.ErrExist) {
		prompt := &survey.Confirm{
			Message: "File(s) already exists. Overwrite?",
		}
		if err = survey.AskOne(prompt, &overwrite); err != nil {
			return errors.WithStack(err)
		}
		if overwrite {
			err = box.PullGlobal(cmd.Context(), overwrite, args[0])
		}
	}
	if err != nil {
		return err
	}

	return installCmdFunc(cmd, runCmdFlags{config: configFlags{path: path}})
}

var globalConfigPath string

func ensureGlobalConfig(cmd *cobra.Command) (string, error) {
	if globalConfigPath != "" {
		return globalConfigPath, nil
	}

	globalConfigPath, err := devbox.GlobalDataPath()
	if err != nil {
		return "", err
	}
	_, err = devbox.InitConfig(globalConfigPath, cmd.ErrOrStderr())
	if err != nil {
		return "", err
	}
	return globalConfigPath, nil
}

func setGlobalConfigForDelegatedCommands(
	globalCmd *cobra.Command,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		globalPath, err := ensureGlobalConfig(cmd)
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
	path, err := ensureGlobalConfig(cmd)
	if err != nil {
		return errors.WithStack(err)
	}

	box, err := devbox.Open(path, cmd.ErrOrStderr())
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
