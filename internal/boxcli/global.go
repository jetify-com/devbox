// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/impl"
	"go.jetpack.io/devbox/internal/nix"
)

func globalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "global",
		Short:  "Manage global devbox packages",
		Hidden: true,
	}

	cmd.AddCommand(globalAddCmd())
	cmd.AddCommand(globalListCmd())
	cmd.AddCommand(globalPullCmd())
	cmd.AddCommand(globalRemoveCmd())
	cmd.AddCommand(globalShellenvCmd())

	return cmd
}

func globalAddCmd() *cobra.Command {
	command := &cobra.Command{
		Use:     "add <pkg>...",
		Short:   "Add a new global package",
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"Usage: %s\n\n%s\n",
					cmd.UseLine(),
					toSearchForPackages,
				)
				return nil
			}
			err := addGlobalCmdFunc(cmd, args)
			if errors.Is(err, nix.ErrPackageNotFound) {
				return usererr.New("%s\n\n%s", err, toSearchForPackages)
			}
			return err
		},
	}

	return command
}

func globalRemoveCmd() *cobra.Command {
	command := &cobra.Command{
		Use:     "rm <pkg>...",
		Aliases: []string{"remove"},
		Short:   "Remove a global package",
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"Usage: %s\n\n%s\n",
					cmd.UseLine(),
					toSearchForPackages,
				)
				return nil
			}
			return removeGlobalCmdFunc(cmd, args)
		},
	}

	return command
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
	return &cobra.Command{
		Use:     "pull <file> | <url>",
		Short:   "Pulls a global config from a file or URL",
		Long:    "Pulls a global config from a file or URL. URLs must be prefixed with 'http://' or 'https://'.",
		PreRunE: ensureNixInstalled,
		RunE:    pullGlobalCmdFunc,
		Args:    cobra.ExactArgs(1),
	}
}

func globalShellenvCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "shellenv",
		Short: "Print shell commands that add global Devbox packages to your PATH",
		Run: func(*cobra.Command, []string) {
			fmt.Print(impl.GenerateShellEnv())
		},
	}
}

func addGlobalCmdFunc(cmd *cobra.Command, args []string) error {
	path, err := devbox.GlobalDataPath()
	if err != nil {
		return err
	}
	if _, err := devbox.InitConfig(path, cmd.ErrOrStderr()); err != nil {
		return errors.WithStack(err)
	}
	box, err := devbox.Open(path, cmd.ErrOrStderr())
	if err != nil {
		return errors.WithStack(err)
	}

	return box.AddGlobal(args...)
}

func removeGlobalCmdFunc(cmd *cobra.Command, args []string) error {
	path, err := devbox.GlobalDataPath()
	if err != nil {
		return err
	}
	if _, err := devbox.InitConfig(path, cmd.ErrOrStderr()); err != nil {
		return errors.WithStack(err)
	}
	box, err := devbox.Open(path, cmd.ErrOrStderr())
	if err != nil {
		return errors.WithStack(err)
	}

	return box.RemoveGlobal(args...)
}

func listGlobalCmdFunc(cmd *cobra.Command, args []string) error {
	path, err := devbox.GlobalDataPath()
	if err != nil {
		return err
	}
	if _, err := devbox.InitConfig(path, cmd.ErrOrStderr()); err != nil {
		return errors.WithStack(err)
	}
	box, err := devbox.Open(path, cmd.OutOrStdout())
	if err != nil {
		return errors.WithStack(err)
	}
	return box.PrintGlobalList()
}

func pullGlobalCmdFunc(cmd *cobra.Command, args []string) error {
	path, err := devbox.GlobalDataPath()
	if err != nil {
		return err
	}
	if _, err := devbox.InitConfig(path, cmd.ErrOrStderr()); err != nil {
		return errors.WithStack(err)
	}
	box, err := devbox.Open(path, cmd.OutOrStdout())
	if err != nil {
		return errors.WithStack(err)
	}
	return box.PullGlobal(args[0])
}
