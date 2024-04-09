// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/devbox"
	"go.jetpack.io/devbox/internal/devbox/devopt"
	"go.jetpack.io/devbox/internal/redact"
)

type runCmdFlags struct {
	envFlag
	config      configFlags
	pure        bool
	listScripts bool
}

func runCmd() *cobra.Command {
	flags := runCmdFlags{}
	command := &cobra.Command{
		Use:   "run [<script> | <cmd>]",
		Short: "Run a script or command in a shell with access to your packages",
		Long: "Start a new shell and runs your script or command in it, exiting when done.\n\n" +
			"The script must be defined in `devbox.json`, or else it will be interpreted as an " +
			"arbitrary command. You can pass arguments to your script or command. Everything " +
			"after `--` will be passed verbatim into your command (see examples).\n\n",
		Example: "\nRun a command directly:\n\n  devbox add cowsay\n  devbox run cowsay hello\n  " +
			"devbox run -- cowsay -d hello\n\nRun a script (defined as `\"moo\": \"cowsay moo\"`) " +
			"in your devbox.json:\n\n  devbox run moo",
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScriptCmd(cmd, args, flags)
		},
	}

	flags.envFlag.register(command)
	flags.config.register(command)
	command.Flags().BoolVar(
		&flags.pure, "pure", false, "if this flag is specified, devbox runs the script in an isolated environment inheriting almost no variables from the current environment. A few variables, in particular HOME, USER and DISPLAY, are retained.")
	command.Flags().BoolVarP(
		&flags.listScripts, "list", "l", false, "list all scripts defined in devbox.json")

	command.ValidArgs = listScripts(command, flags)

	return command
}

func listScripts(cmd *cobra.Command, flags runCmdFlags) []string {
	box, err := devbox.Open(&devopt.Opts{
		Dir:            flags.config.path,
		Environment:    flags.config.environment,
		Stderr:         cmd.ErrOrStderr(),
		Pure:           flags.pure,
		IgnoreWarnings: true,
	})
	if err != nil {
		debug.Log("failed to open devbox: %v", err)
		return nil
	}

	return box.ListScripts()
}

func runScriptCmd(cmd *cobra.Command, args []string, flags runCmdFlags) error {
	if len(args) == 0 || flags.listScripts {
		scripts := listScripts(cmd, flags)
		if len(scripts) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "no scripts defined in devbox.json")
			return nil
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Available scripts:")
		for _, p := range scripts {
			fmt.Fprintf(cmd.OutOrStdout(), "* %s\n", p)
		}
		return nil
	}

	path, script, scriptArgs, err := parseScriptArgs(args, flags)
	if err != nil {
		return redact.Errorf("error parsing script arguments: %w", err)
	}
	debug.Log("script: %s", script)
	debug.Log("script args: %v", scriptArgs)

	env, err := flags.Env(path)
	if err != nil {
		return err
	}

	// Check the directory exists.
	box, err := devbox.Open(&devopt.Opts{
		Dir:         path,
		Environment: flags.config.environment,
		Stderr:      cmd.ErrOrStderr(),
		Pure:        flags.pure,
		Env:         env,
	})
	if err != nil {
		return redact.Errorf("error reading devbox.json: %w", err)
	}

	if err := box.RunScript(cmd.Context(), script, scriptArgs); err != nil {
		return redact.Errorf("error running script %q in Devbox: %w", script, err)
	}
	return nil
}

func parseScriptArgs(args []string, flags runCmdFlags) (string, string, []string, error) {
	if len(args) == 0 {
		// this should never happen because cobra should prevent it, but it's better to be defensive.
		return "", "", nil, usererr.New("no command or script provided")
	}

	script := args[0]
	scriptArgs := args[1:]

	return flags.config.path, script, scriptArgs, nil
}

func wrapArgsForRun(rootCmd *cobra.Command, args []string) []string {
	// if the first argument is not "run", we don't need to do anything. If there
	// are 2 or fewer arguments, we also don't need to do anything because there
	// are no flags after a non-run non-flag arg.
	// IMPROVEMENT: technically users can pass a flag before the subcommand "run"
	if len(args) <= 2 || args[0] != "run" || slices.Contains(args, "--") {
		return args
	}

	cmd, found := lo.Find(
		rootCmd.Commands(),
		func(item *cobra.Command) bool { return item.Name() == "run" },
	)
	if !found {
		return args
	}
	_ = cmd.InheritedFlags() // bug in cobra requires this to be called to ensure flags contains inherited flags.
	runFlags := cmd.Flags()
	// typical args can be of the form:
	// run --flag1 val1 -f val2 --flag3=val3 --bool-flag python --version
	// We handle each different type of flag
	// (flag with equals, long-form, short-form, and defaulted flags)
	// Note that defaulted does not mean initial value, it only means flags
	// that don't require a value.
	// For example, --bool-flag has NoOptDefVal set to "true".
	i := 1
	for i < len(args) {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			// We found and argument that is not part of the flags, so we can stop
			// This inserts a "--" before the first non-flag argument
			// Turning
			// run --flag1 val1 command --flag2 val2
			// into
			// run --flag1 val1 -- command --flag2 val2
			return append(args[:i+1], append([]string{"--"}, args[i+1:]...)...)
		}

		if strings.HasPrefix(arg, "-") && strings.Contains(arg, "=") {
			// This is a flag with an equals sign, so we can skip it
			i++
			continue
		}

		var flag *pflag.Flag
		if strings.HasPrefix(arg, "--") {
			flag = runFlags.Lookup(strings.TrimLeft(arg, "-"))
		} else {
			flag = runFlags.ShorthandLookup(strings.TrimLeft(arg, "-"))
		}
		if flag == nil {
			// found an invalid flag, just return args as-is
			return args
		}
		if flag.NoOptDefVal == "" {
			// This is a non-boolean flag, e.g. --flag1 val1
			i += 2
		} else {
			// This is a boolean flag, e.g. --bool-flag
			i++
		}
	}

	// This means there is no non-flag command. Just return as is.
	return args
}
