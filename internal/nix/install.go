package nix

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

//go:embed install.sh
var installScript string

func Install(writer io.Writer) error {
	r, w, err := os.Pipe()
	if err != nil {
		return errors.WithStack(err)
	}
	defer r.Close()

	cmd := exec.Command("sh", "-c", installScript)
	// Attach stdout but no stdin. This makes the command run in non-TTY mode
	// which skips the interactive prompts.
	// We could attach stderr? but the stdout prompt is pretty useful.
	cmd.Stdin = nil
	cmd.Stdout = w
	cmd.Stderr = w

	fmt.Fprintln(writer, "Installing Nix. This may require sudo access.")
	err = cmd.Start()
	w.Close()
	if err != nil {
		return errors.WithStack(err)
	}

	done := make(chan struct{})
	go func() {
		_, err := io.Copy(os.Stdout, r)
		if err != nil {
			fmt.Fprintln(writer, err)
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
