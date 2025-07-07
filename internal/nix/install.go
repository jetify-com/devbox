// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"

	"go.jetify.com/devbox/internal/boxcli/usererr"
	"go.jetify.com/devbox/internal/cmdutil"
	"go.jetify.com/devbox/internal/fileutil"
	"go.jetify.com/devbox/nix"
)

func BinaryInstalled() bool {
	return cmdutil.Exists("nix")
}

func dirExistsAndIsNotEmpty(dir string) bool {
	empty, err := fileutil.IsDirEmpty(dir)
	return err == nil && !empty
}

var ensured = false

func Ensured() bool {
	return ensured
}

func EnsureNixInstalled(ctx context.Context, writer io.Writer, withDaemonFunc func() *bool) (err error) {
	ensured = true
	defer func() {
		if err != nil {
			return
		}

		// ensure minimum nix version installed
		if !nix.AtLeast(MinVersion) {
			err = usererr.New(
				"Devbox requires nix of version >= %s. Your version is %s. "+
					"Please upgrade nix and try again.\n",
				MinVersion,
				nix.Version(),
			)
			return
		}
	}()

	if BinaryInstalled() {
		return nil
	}
<<<<<<< HEAD
	if dirExistsAndIsNotEmpty() {
=======
	if dirExistsAndIsNotEmpty("/nix") {
>>>>>>> ascknx/main
		if _, err = SourceProfile(); err != nil {
			return err
		} else if BinaryInstalled() {
			return nil
		}

		return usererr.New(
			"We found a /nix directory but nix binary is not in your PATH and we " +
				"were not able to find it in the usual locations. Your nix installation " +
				"might be broken. If restarting your terminal or reinstalling nix " +
				"doesn't work, please create an issue at " +
				"https://github.com/jetify-com/devbox/issues",
		)
	}

	color.Yellow("\nNix is not installed. Devbox will attempt to install it.\n\n")

	installer := nix.Installer{}
	if isatty.IsTerminal(os.Stdout.Fd()) {
		color.Yellow("Press enter to continue or ctrl-c to exit.\n")
		fmt.Scanln() //nolint:errcheck

		spinny := spinner.New(spinner.CharSets[11], 100*time.Millisecond, spinner.WithWriter(writer))
		spinny.Suffix = " Downloading the Nix installer..."
		spinny.Start()
		defer spinny.Stop() // reset the terminal in case of a panic

		err = installer.Download(ctx)
		if err != nil {
			return err
		}
		spinny.Stop()
	} else {
		fmt.Fprint(writer, "Downloading the Nix installer...")
		err = installer.Download(ctx)
		if err != nil {
			fmt.Fprintln(writer)
			return err
		}
		fmt.Fprintln(writer, " done.")
	}
	err = installer.Run(ctx)
	if err != nil {
		return err
	}

	fmt.Fprintln(writer, "Nix installed successfully. Devbox is ready to use!")
	return nil
}
