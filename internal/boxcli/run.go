// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/debug"
)

type runCmdFlags struct {
	config configFlags
}

func RunCmd() *cobra.Command {
	flags := runCmdFlags{}
	command := &cobra.Command{
		Use:   "run [<script> | <cmd>]",
		Short: "Runs a script or command in a shell with access to your packages",
		Long: "Starts a new shell and runs your script or command in it, exiting when done.\n\n" +
			"The script must be defined in `devbox.json`, or else it will be interpreted as an " +
			"arbitrary command. You can pass arguments to your script or command. Everything " +
			"after `--` will be passed verbatim into your command (see examples).\n\n",
		Example: "\nRun a command directly:\n\n  devbox add cowsay\n  devbox run cowsay hello\n  " +
			"devbox run -- cowsay -d hello\n\nRun a script (defined as `\"moo\": \"cowsay moo\"`) " +
			"in your devbox.json:\n\n  devbox run moo",
		Args:    cobra.MinimumNArgs(1),
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScriptCmd(cmd, args, flags)
		},
	}

	flags.config.register(command)

	command.ValidArgs = listScripts(command, flags)

	return command
}

func listScripts(cmd *cobra.Command, flags runCmdFlags) []string {
	path, err := configPathFromUser([]string{}, &flags.config)
	if err != nil {
		debug.Log("failed to get config path from user: %v", err)
		return nil
	}

	box, err := devbox.Open(path, cmd.ErrOrStderr())
	if err != nil {
		debug.Log("failed to open devbox: %v", err)
		return nil
	}

	return box.ListScripts()
}

func runScriptCmd(cmd *cobra.Command, args []string, flags runCmdFlags) error {
	path, script, scriptArgs, err := parseScriptArgs(args, flags)
	if err != nil {
		return err
	}
	debug.Log("script: %s", script)
	debug.Log("script args: %v", scriptArgs)

	// Check the directory exists.
	box, err := devbox.Open(path, cmd.ErrOrStderr())
	if err != nil {
		return errors.WithStack(err)
	}

	return box.RunScript(script, scriptArgs)
}

func parseScriptArgs(args []string, flags runCmdFlags) (string, string, []string, error) {
	path, err := configPathFromUser([]string{}, &flags.config)
	if err != nil {
		return "", "", nil, err
	}

	if len(args) == 0 {
		// this should never happen because cobra should prevent it, but it's better to be defensive.
		return "", "", nil, usererr.New("no command or script provided")
	}

	script := args[0]
	scriptArgs := args[1:]

	return path, script, scriptArgs, nil
}
