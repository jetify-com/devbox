// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"log/slog"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"go.jetify.com/devbox/internal/boxcli/multi"
	"go.jetify.com/devbox/internal/boxcli/usererr"
	"go.jetify.com/devbox/internal/devbox"
	"go.jetify.com/devbox/internal/devbox/devopt"
	"go.jetify.com/devbox/internal/redact"
	"go.jetify.com/devbox/internal/ux"
)

type runCmdFlags struct {
	envFlag
	config       configFlags
	omitNixEnv   bool
	pure         bool
	listScripts  bool
	recomputeEnv bool
	allProjects  bool
}

// runFlagDefaults are the flag default values that differ
// from the `devbox` command versus `devbox global` command.
type runFlagDefaults struct {
	omitNixEnv bool
}

func runCmd(defaults runFlagDefaults) *cobra.Command {
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
	command.Flags().BoolVar(
		&flags.omitNixEnv, "omit-nix-env", defaults.omitNixEnv,
		"shell environment will omit the env-vars from print-dev-env",
	)
	_ = command.Flags().MarkHidden("omit-nix-env")
	command.Flags().BoolVar(&flags.recomputeEnv, "recompute", true, "recompute environment if needed")
	command.Flags().BoolVar(
		&flags.allProjects,
		"all-projects",
		false,
		"run command in all projects in the working directory, recursively. If command is not found in any project, it will be skipped.",
	)

	command.ValidArgs = listScripts(command, flags)

	return command
}

func listScripts(cmd *cobra.Command, flags runCmdFlags) []string {
	path := flags.config.path

	// Special code path for shell completion.
	// Landau: I'm not entirely sure why:
	// * Flags need to be parsed again
	// * cmd.Flag("config") contains the correct value, but flags.config.path is empty
	// Give my low confidence, I'm making this a very narrow code path.
	if path == "" && slices.Contains(os.Args, "__complete") {
		_ = cmd.ParseFlags(os.Args)
		if flag := cmd.Flag("config"); flag != nil && flag.Value != nil {
			path = flag.Value.String()
		}
	}

	devboxOpts := &devopt.Opts{
		Dir:            path,
		Environment:    flags.config.environment,
		Stderr:         cmd.ErrOrStderr(),
		IgnoreWarnings: true,
	}

	if flags.allProjects {
		boxes, err := multi.Open(devboxOpts)
		if err != nil {
			slog.Error("failed to open devbox", "err", err)
			return nil
		}
		scripts := []string{}
		for _, box := range boxes {
			scripts = append(scripts, box.ListScripts()...)
		}
		sort.Strings(scripts)
		return lo.Uniq(scripts)
	}
	box, err := devbox.Open(devboxOpts)
	if err != nil {
		slog.Error("failed to open devbox", "err", err)
		return nil
	}
	return box.ListScripts()
}

func runScriptCmd(cmd *cobra.Command, args []string, flags runCmdFlags) error {
	ctx := cmd.Context()
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
	slog.Debug("run script", "script", script, "args", scriptArgs)

	env, err := flags.Env(path)
	if err != nil {
		return err
	}

	boxes := []*devbox.Devbox{}
	devboxOpts := &devopt.Opts{
		Dir:         path,
		Env:         env,
		Environment: flags.config.environment,
		Stderr:      cmd.ErrOrStderr(),
	}

	if flags.allProjects {
		boxes, err = multi.Open(devboxOpts)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		box, err := devbox.Open(devboxOpts)
		if err != nil {
			return redact.Errorf("error reading devbox.json: %w", err)
		}
		boxes = append(boxes, box)
	}

	envOpts := devopt.EnvOptions{
		Hooks: devopt.LifecycleHooks{
			OnStaleState: func() {
				if !flags.recomputeEnv {
					ux.FHidableWarning(
						ctx,
						cmd.ErrOrStderr(),
						devbox.StateOutOfDateMessage,
						"with --recompute=true",
					)
				}
			},
		},
		OmitNixEnv:    flags.omitNixEnv,
		Pure:          flags.pure,
		SkipRecompute: !flags.recomputeEnv,
	}

	if flags.allProjects {
		boxes = lo.Filter(boxes, func(box *devbox.Devbox, _ int) bool {
			return slices.Contains(box.ListScripts(), script)
		})
	}

	for _, box := range boxes {
		ux.Finfof(
			cmd.ErrOrStderr(),
			"Running script %q on %s\n",
			script,
			box.ProjectDir(),
		)
		if err := box.RunScript(ctx, envOpts, script, scriptArgs); err != nil {
			return redact.Errorf("error running script %q in Devbox: %w", script, err)
		}
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
