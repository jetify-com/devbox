package nix

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
)

const rootError = "warning: installing Nix as root is not supported by this script!"

// Install runs the install script for Nix. daemon has 3 states
// nil is unset. false is --no-daemon. true is --daemon.
func Install(writer io.Writer, daemon *bool) error {
	r, w, err := os.Pipe()
	if err != nil {
		return errors.WithStack(err)
	}
	defer r.Close()

	installScript := "curl -L https://nixos.org/nix/install | sh -s"
	if daemon != nil {
		if *daemon {
			installScript += " -- --daemon"
		} else {
			installScript += " -- --no-daemon"
		}
	}

	fmt.Fprintf(writer, "Installing nix with: %s\nThis may require sudo access.\n", installScript)

	cmd := exec.Command("sh", "-c", installScript)
	// Attach stdout but no stdin. This makes the command run in non-TTY mode
	// which skips the interactive prompts.
	// We could attach stderr? but the stdout prompt is pretty useful.
	cmd.Stdin = nil
	cmd.Stdout = w
	cmd.Stderr = w

	err = cmd.Start()
	w.Close()
	if err != nil {
		return errors.WithStack(err)
	}

	done := make(chan struct{})
	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(io.MultiWriter(&buf, os.Stdout), r)
		if err != nil {
			fmt.Fprintln(writer, err)
		}

		if strings.Contains(buf.String(), rootError) {
			color.New(color.FgYellow).Fprintln(
				writer,
				"If installing nix as root, consider using the --daemon flag to install in multi-user mode.",
			)
		}
		close(done)
	}()

	<-done
	return errors.WithStack(cmd.Wait())
}

func BinaryInstalled() bool {
	_, err := exec.LookPath("nix-shell")
	return err == nil
}

func DirExists() bool {
	_, err := os.Stat("/nix")
	return err == nil
}
