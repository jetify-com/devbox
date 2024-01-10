package nix

import (
	"context"
	"os/exec"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/debug"
)

func Build(ctx context.Context, flags []string, installables ...string) error {
	// --impure is required for allowUnfreeEnv to work.
	cmd := commandContext(ctx, "build", "--impure")
	cmd.Args = append(cmd.Args, flags...)
	cmd.Args = append(cmd.Args, installables...)
	// We need to allow Unfree packages to be installed. We choose to not also add os.Environ() to the environment
	// to keep the command as pure as possible, even though we must pass --impure to nix build.
	cmd.Env = allowUnfreeEnv([]string{})

	debug.Log("Running cmd: %s\n", cmd)
	_, err := cmd.Output()
	if err != nil {
		if exitErr := (&exec.ExitError{}); errors.As(err, &exitErr) {
			debug.Log("Nix build exit code: %d, output: %s\n", exitErr.ExitCode(), exitErr.Stderr)
		}
		return err
	}
	return nil
}
