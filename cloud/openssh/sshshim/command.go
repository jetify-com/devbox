package sshshim

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"go.jetpack.io/devbox/boxcli/midcobra"
	"go.jetpack.io/devbox/build"
	"go.jetpack.io/devbox/debug"
)

func Execute(ctx context.Context, args []string) int {
	defer debug.Recover()

	exe := midcobra.New(&sshshimCommand{})
	exe.AddMiddleware(midcobra.Sentry(&midcobra.SentryOpts{
		AppName:    "devbox-sshshim",
		AppVersion: build.Version,
		SentryDSN:  build.SentryDSN,
	}))
	return exe.Execute(ctx, args)
}

// sshshimCommand implements midcobra.Command
var _ midcobra.Command = (*sshshimCommand)(nil)

type sshshimCommand struct {
	args []string
}

func (cmd *sshshimCommand) ExecuteContext(ctx context.Context) error {
	return execute(cmd.args)
}

func (cmd *sshshimCommand) SetArgs(args []string) {
	cmd.args = args
}
func (cmd *sshshimCommand) ShouldTraverseChildren() bool {
	panic("not implemented")
}
func (cmd *sshshimCommand) Traverse(args []string) (*cobra.Command, []string, error) {
	panic("not implemented")
}
func (cmd *sshshimCommand) Find(args []string) (*cobra.Command, []string, error) {
	panic("not implemented")
}
func (cmd *sshshimCommand) Flag(name string) *flag.Flag {
	panic("not implemented")
}

func execute(args []string) error {
	EnableDebug() // Always enable for now.
	debug.Log("os.Args: %v", args)

	if alive, err := ensureLiveVMOrTerminateMutagenSessions(args[1:]); err != nil {
		debug.Log("ensureLiveVMOrTerminateMutagenSessions error: %v", err)
		fmt.Fprintf(os.Stderr, "%v", err)
		return err
	} else if !alive {
		return nil
	}

	if err := invokeSSHOrSCPCommand(args); err != nil {
		debug.Log("InvokeSSHorSCPCommand error: %v", err)
		fmt.Fprintf(os.Stderr, "%v", err)
		return err
	}
	return nil
}
