package nix

import (
	"context"
	"io"
	"os"

	"go.jetpack.io/devbox/internal/debug"
)

type BuildArgs struct {
	AllowInsecure bool
	Flags         []string
	Writer        io.Writer
}

func Build(ctx context.Context, args *BuildArgs, installables ...string) error {
	// --impure is required for allowUnfreeEnv/allowInsecureEnv to work.
	cmd := commandContext(ctx, "build", "--impure")
	cmd.Args = append(cmd.Args, args.Flags...)
	cmd.Args = append(cmd.Args, installables...)
	cmd.Env = allowUnfreeEnv(os.Environ())
	if args.AllowInsecure {
		debug.Log("Setting Allow-insecure env-var\n")
		cmd.Env = allowInsecureEnv(cmd.Env)
	}

	// If nix build runs as tty, the output is much nicer. If we ever
	// need to change this to our own writers, consider that you may need
	// to implement your own nicer output. --print-build-logs flag may be useful.
	cmd.Stdin = os.Stdin
	cmd.Stdout = args.Writer
	cmd.Stderr = args.Writer

	debug.Log("Running cmd: %s\n", cmd)
	return cmd.Run()
}
