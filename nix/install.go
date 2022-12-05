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
	cmd.Stdin = nil
	cmd.Stdout = w
	cmd.Stderr = nil

	fmt.Println("Installing Nix. This will require sudo access.")
	if err = cmd.Start(); err != nil {
		return errors.WithStack(err)
	}

	go io.Copy(os.Stdout, r)
	return errors.WithStack(cmd.Wait())
}
