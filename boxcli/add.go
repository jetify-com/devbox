// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
)

func AddCmd() *cobra.Command {
	command := &cobra.Command{
		Use:               "add <pkg>...",
		Short:             "Add a new package to your devbox",
		Args:              cobra.MinimumNArgs(1),
		PersistentPreRunE: nixShellPersistentPreRunE,
		RunE:              addCmdFunc(),
	}

	return command
}

func addCmdFunc() runFunc {
	return func(cmd *cobra.Command, args []string) error {
		box, err := devbox.Open(".")
		if err != nil {
			return errors.WithStack(err)
		}

		if err = box.Add(args...); err != nil {
			return err
		}

		if err := box.Generate(); err != nil {
			return err
		}

		fmt.Print("Installing nix packages. This may take a while...")
		if err = installDevPackages(box.SourceDir()); err != nil {
			fmt.Println()
			return err
		}
		fmt.Println("done.")

		if isDevboxShellEnabled() {
			successMsg := fmt.Sprintf("%s is now installed.", args[0])
			if len(args) > 1 {
				successMsg = fmt.Sprintf("%s are now installed.", strings.Join(args, ", "))
			}
			fmt.Print(successMsg)
			fmt.Println(" Run `hash -r` to ensure your shell is updated.")
		}

		return nil
	}
}
