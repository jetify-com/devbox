package nix

import (
	_ "embed"
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

//go:embed install.sh
var installScript string

func Install() error {
	cmd := exec.Command("sh", "-c", installScript)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return errors.WithStack(cmd.Run())
}
