package nix

import (
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/boxcli/featureflag"
	"go.jetpack.io/devbox/boxcli/usererr"
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

	cmd := exec.Command("sudo", "sh", "-c", installScript)
	// Attach stdout but no stdin. This makes the command run in non-TTY mode
	// which skips the interactive prompts.
	// We could attach stderr? but the stdout prompt is pretty useful.
	cmd.Stdin = nil
	cmd.Stdout = w
	cmd.Stderr = nil

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
	_, err := exec.LookPath("nix-shell")
	if err == nil {
		return nil
	}

	if featureflag.NixInstaller.Enabled() {
		color.Yellow("\nNix is not installed. Devbox will attempt to install it. \n\nPress enter to continue.\n")
		_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
		if err = Install(); err != nil {
			return err
		}
		return usererr.NewWarning("Nix requires reopening terminal to function correctly. Please open new terminal and try again.")
	}
	return usererr.New("could not find nix in your PATH\nInstall nix by following the instructions at https://nixos.org/download.html and make sure you've set up your PATH correctly")
}
