package nix

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/debug"
)

func Build(ctx context.Context, flags []string, installables ...string) error {
	// --impure is required for allowUnfreeEnv to work.
	cmd := commandContext(ctx, "build", "--impure")
	cmd.Args = append(cmd.Args, flags...)
	cmd.Args = append(cmd.Args, installables...)
	cmd.Env = allowUnfreeEnv(os.Environ())

	debug.Log("Running cmd: %s\n", cmd)
	_, err := cmd.Output()
	if err != nil {
		if exitErr := (&exec.ExitError{}); errors.As(err, &exitErr) {
			debug.Log("Nix build exit code: %d, output: %s\n", exitErr.ExitCode(), exitErr.Stderr)
			return fmt.Errorf("nix build exit code: %d, output: %s, err: %w", exitErr.ExitCode(), exitErr.Stderr, err)
		}
		return err
	}
	return nil
}
