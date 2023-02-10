// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/debug"
)

type runCmdFlags struct {
	config configFlags
}

func RunCmd() *cobra.Command {
	flags := runCmdFlags{}
	command := &cobra.Command{
		Use:     "run <script>",
		Short:   "Starts a new devbox shell and runs the target script",
		Long:    "Starts a new interactive shell and runs your target script in it. The shell will exit once your target script is completed or when it is terminated via CTRL-C. Scripts can be defined in your `devbox.json`",
		Args:    cobra.MinimumNArgs(1),
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScriptCmd(cmd, args, flags)
		},
	}

	flags.config.register(command)

	return command
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

	if featureflag.UnifiedEnv.Enabled() {
		err = box.RunScript(script, scriptArgs)
	} else {
		if devbox.IsDevboxShellEnabled() {
			err = box.RunScriptInShell(script)
		} else {
			err = box.RunScript(script, scriptArgs)
		}
	}
	return err
}

func parseScriptArgs(args []string, flags runCmdFlags) (string, string, []string, error) {
	path, err := configPathFromUser([]string{}, &flags.config)
	if err != nil {
		return "", "", nil, err
	}

	script := ""
	var scriptArgs []string
	if len(args) >= 1 {
		script = args[0]
		scriptArgs = args[1:]
	} else {
		// this should never happen because cobra should prevent it, but it's better to be defensive.
		return "", "", nil, usererr.New("no command or script provided")
	}

	return path, script, scriptArgs, nil
}
