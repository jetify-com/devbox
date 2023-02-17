// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"github.com/pkg/errors"
	"github.com/samber/lo"
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
	longHelp := "Starts a new shell and runs your script or command in it, exiting when done.\n\n" +
		"The script must be defined in `devbox.json`, or else it will be interpreted as an " +
		"arbitrary command. You can pass arguments to your script or command. Everything " +
		"after `--` will be passed verbatim into your command (see examples).\n\n"
	shortHelp := "Runs a script or command in a shell with access to your packages"
	example := "\nRun a command directly:\n\n  devbox add cowsay\n  devbox run cowsay hello\n  " +
		"devbox run -- cowsay -d hello\n\nRun a script (defined as `\"moo\": \"cowsay moo\"`) " +
		"in your devbox.json:\n\n  devbox run moo"
	if featureflag.UnifiedEnv.Disabled() {
		shortHelp = "Starts a new devbox shell and runs the target script"
		longHelp = "Starts a new interactive shell and runs your target script in it. The shell will " +
			"exit once your target script is completed or when it is terminated via CTRL-C. " +
			"Scripts can be defined in your `devbox.json`"
		example = ""
	}
	flags := runCmdFlags{}
	command := &cobra.Command{
		Use:     lo.Ternary(featureflag.UnifiedEnv.Enabled(), "run [<script> | <cmd>]", "run <script>"),
		Short:   shortHelp,
		Long:    longHelp,
		Example: example,
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
