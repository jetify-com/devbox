package boxcli

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/axiom/opensource/devbox"
)

func PlanCmd() *cobra.Command {
	command := &cobra.Command{
		Use:  "plan [<dir>]",
		Args: cobra.MaximumNArgs(1),
		RunE: runPlanCmd,
	}
	return command
}

func runPlanCmd(cmd *cobra.Command, args []string) error {
	path := pathArg(args)

	// Check the directory exists.
	box, err := devbox.Open(path)
	if err != nil {
		return errors.WithStack(err)
	}

	plan := box.Plan()
	fmt.Println(plan)
	return nil
}
