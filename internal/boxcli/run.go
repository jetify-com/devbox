// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"os/exec"
	"sort"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/boxcli/writer"
	"go.jetpack.io/devbox/internal/nix"
	"golang.org/x/exp/slices"
)

type runCmdFlags struct {
	config configFlags
}

func RunCmd() *cobra.Command {
	flags := runCmdFlags{}
	command := &cobra.Command{
		Use:               "run <script>",
		Short:             "Starts a new devbox shell and runs the target script",
		Long:              "Starts a new interactive shell and runs your target script in it. The shell will exit once your target script is completed or when it is terminated via CTRL-C. Scripts can be defined in your `devbox.json`",
		Args:              cobra.MaximumNArgs(1),
		PersistentPreRunE: nix.EnsureInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScriptCmd(cmd, args, flags)
		},
	}

	flags.config.register(command)

	return command
}

func runScriptCmd(cmd *cobra.Command, args []string, flags runCmdFlags) error {

	w := writer.New(cmd)
	path, script, err := parseScriptArgs(args, flags)
	if err != nil {
		return err
	}

	// Check the directory exists.
	box, err := devbox.Open(path, w)
	if err != nil {
		return errors.WithStack(err)
	}

	// Validate script exists.
	scripts := box.ListScripts()
	sort.Slice(scripts, func(i, j int) bool { return scripts[i] < scripts[j] })
	if script == "" || !slices.Contains(scripts, script) {
		return errors.Errorf("no script found with name \"%s\". "+
			"Here's a list of the existing scripts in devbox.json: %v", script, scripts)
	}

	if devbox.IsDevboxShellEnabled() {
		err = box.RunScriptInShell(script)
	} else {
		err = box.RunScript(script)
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil
	}
	return err
}

func parseScriptArgs(args []string, flags runCmdFlags) (string, string, error) {
	path, err := configPathFromUser([]string{}, &flags.config)
	if err != nil {
		return "", "", err
	}

	script := ""
	if len(args) == 1 {
		script = args[0]
	}

	return path, script, nil
}
