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

func RemoveCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "rm <pkg>...",
		Short: "Remove a package from your devbox",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runRemoveCmd,
	}
	return command
}

func runRemoveCmd(cmd *cobra.Command, args []string) error {
	box, err := devbox.Open(".")
	if err != nil {
		return errors.WithStack(err)
	}

	if err = box.Remove(args...); err != nil {
		return err
	}

	if err := box.Generate(); err != nil {
		return err
	}

	fmt.Print("Uninstalling nix packages. This may take a while...")
	// We need to reinstall the packages
	if err = installDevPackages(box.SourceDir()); err != nil {
		fmt.Println()
		return err
	}
	fmt.Println("done.")

	if isDevboxShellEnabled() {
		successMsg := fmt.Sprintf("%s is now removed.", args[0])
		if len(args) > 1 {
			successMsg = fmt.Sprintf("%s are now removed.", strings.Join(args, ", "))
		}
		fmt.Print(successMsg)
		fmt.Println(" Run `hash -r` to ensure your shell is updated.")
	}

	return nil
}
