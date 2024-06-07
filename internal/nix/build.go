package nix

import (
	"context"
	"io"
	"os"
	"strings"

	"go.jetpack.io/devbox/internal/debug"
)

type BuildArgs struct {
	AllowInsecure     bool
	Env               []string
	ExtraSubstituters []string
	Flags             []string
	Writer            io.Writer
}

func Build(ctx context.Context, args *BuildArgs, installables ...string) error {
	defer debug.FunctionTimer().End()
	// --impure is required for allowUnfreeEnv/allowInsecureEnv to work.
	cmd := command("build", "--impure")
	cmd.Args = appendArgs(cmd.Args, args.Flags)
	cmd.Args = appendArgs(cmd.Args, installables)
	// Adding extra substituters only here to be conservative, but this could also
	// be added to ExperimentalFlags() in the future.
	if len(args.ExtraSubstituters) > 0 {
		cmd.Args = append(cmd.Args,
			"--extra-substituters",
			strings.Join(args.ExtraSubstituters, " "),
		)
	}
	cmd.Env = append(allowUnfreeEnv(os.Environ()), args.Env...)
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
	return cmd.Run(ctx)
}
