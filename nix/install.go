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
