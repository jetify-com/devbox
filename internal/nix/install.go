package nix

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
)

//go:embed install.sh
var installScript string

func Install() error {
	r, w, err := os.Pipe()
	if err != nil {
		return errors.WithStack(err)
	}
	defer r.Close()
	defer w.Close()

	cmd := exec.Command("sh", "-c", installScript, "--", "--daemon")
	// Attach stdout but no stdin. This makes the command run in non-TTY mode
	// which skips the interactive prompts.
	// We could attach stderr? but the stdout prompt is pretty useful.
	cmd.Stdin = nil
	cmd.Stdout = w
	cmd.Stderr = w

	fmt.Println("Installing Nix. This will require sudo access.")
	if err = cmd.Start(); err != nil {
		return errors.WithStack(err)
	}

	go func() {
		_, err := io.Copy(os.Stdout, r)
		if err != nil {
			fmt.Println(err)
		}
	}()

	return errors.WithStack(cmd.Wait())
}

func EnsureInstalled(cmd *cobra.Command, args []string) error {
	if nixBinaryInstalled() {
		return nil
	}
	if nixDirExists() {
		// TODO: We may be able to patch the rc files to add nix to the path.
		return usererr.New(
			"We found a /nix directory but nix binary is not in your PATH. " +
				"Try restarting your terminal and running devbox again. If after " +
				"restarting you still get this message it's possible nix setup is " +
				"missing from your shell rc file. See " +
				"https://github.com/NixOS/nix/issues/3616#issuecomment-903869569 for " +
				"more details.",
		)
	}

	if featureflag.NixInstaller.Enabled() {
		color.Yellow(
			"\nNix is not installed. Devbox will attempt to install it. " +
				"\n\nPress enter to continue or ctrl-c to exit.\n",
		)
		fmt.Scanln()
		if err := Install(); err != nil {
			return err
		}
		return usererr.NewWarning(
			"Nix requires reopening terminal to function correctly. Please open new" +
				" terminal and try again.",
		)
	}
	return usererr.New(
		"could not find nix in your PATH\nInstall nix by following the " +
			"instructions at https://nixos.org/download.html and make sure you've " +
			"set up your PATH correctly",
	)
}

func nixBinaryInstalled() bool {
	_, err := exec.LookPath("nix-shell")
	return err == nil
}

func nixDirExists() bool {
	_, err := os.Stat("/nix")
	return err == nil
}
