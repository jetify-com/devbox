// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"go.jetify.com/devbox/internal/devbox"
	"go.jetify.com/devbox/internal/skill"
	"go.jetify.com/devbox/internal/ux"
	"go.jetify.com/devbox/pkg/autodetect"
)

type initFlags struct {
	auto   bool
	dryRun bool
	skill  string
}

func initCmd() *cobra.Command {
	flags := &initFlags{}
	command := &cobra.Command{
		Use:   "init [<dir>]",
		Short: "Initialize a directory as a devbox project",
		Long: "Initialize a directory as a devbox project. " +
			"This will create an empty devbox.json in the current directory. " +
			"You can then add packages using `devbox add`",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch flags.skill {
			case "", "none", skillScopeAll, skillScopeProject, skillScopeUser:
			default:
				return fmt.Errorf("invalid --skill %q (must be all, project, user or none)", flags.skill)
			}
			err := runInitCmd(cmd, args, flags)
			path := pathArg(args)
			if path == "" || path == "." {
				path, _ = os.Getwd()
			}
			if errors.Is(err, os.ErrExist) {
				ux.Fwarningf(cmd.ErrOrStderr(), "A devbox config already exists in %q.", path)
				return nil
			}
			if err != nil {
				return err
			}
			if flags.dryRun {
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created devbox.json in %s\n", path)
			fmt.Fprintln(cmd.OutOrStdout(), "Run `devbox add <package>` to add packages, or `devbox shell` to start a dev shell.")
			return initSkill(cmd, path, flags.skill)
		},
	}

	command.Flags().StringVar(
		&flags.skill, "skill", "",
		"install the devbox skill for AI coding agents: all, project, user or none "+
			"(prompts when unset and the terminal is interactive)",
	)

	command.Flags().BoolVar(&flags.auto, "auto", false, "Automatically detect packages to add")
	command.Flags().BoolVar(&flags.dryRun, "dry-run", false, "Dry run for auto mode. Prints the config that would be used")
	_ = command.Flags().MarkHidden("auto")
	_ = command.Flags().MarkHidden("dry-run")

	return command
}

func runInitCmd(cmd *cobra.Command, args []string, flags *initFlags) error {
	path := pathArg(args)

	if flags.auto {
		if flags.dryRun {
			config, err := autodetect.DryRun(cmd.Context(), path)
			if err != nil {
				return errors.WithStack(err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(config))
			return nil
		}
		return autodetect.InitConfig(cmd.Context(), path)
	}

	return devbox.InitConfig(path)
}

// initSkill installs the devbox agent skill after init, either for the scope
// given by --skill or by asking the user.
func initSkill(cmd *cobra.Command, projectDir, scope string) error {
	switch scope {
	case "":
		return promptInstallSkill(cmd, projectDir)
	case "none":
		return nil
	case skillScopeAll:
		return installSkill(cmd, projectDir, []skill.Scope{skill.ScopeProject, skill.ScopeUser})
	case skillScopeProject:
		return installSkill(cmd, projectDir, []skill.Scope{skill.ScopeProject})
	case skillScopeUser:
		return installSkill(cmd, projectDir, []skill.Scope{skill.ScopeUser})
	}
	return nil
}
