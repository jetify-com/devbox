// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"go.jetify.com/devbox/internal/devconfig"
	"go.jetify.com/devbox/internal/envir"
	"go.jetify.com/devbox/internal/skill"
	"go.jetify.com/devbox/internal/ux"
)

const (
	skillScopeProject = "project"
	skillScopeUser    = "user"
	skillScopeAll     = "all"
)

type skillCmdFlags struct {
	config pathFlag
	scope  string
	force  bool
}

func skillCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "skill",
		Short: "Manage the devbox skill for AI coding agents",
		Long: "Manage the devbox skill for AI coding agents.\n\n" +
			"The skill is an Agent Skills SKILL.md (https://agentskills.io) that " +
			"teaches agents such as Claude Code, Codex, Cursor, Copilot and Gemini " +
			"to run commands through devbox in projects that have a devbox.json, " +
			"and how to add packages, use plugins and services, and fix native " +
			"library issues.\n\n" +
			"Project installs write to .agents/skills/devbox and " +
			".claude/skills/devbox in the project root; commit them so every " +
			"agent working in the repo uses devbox. User installs write the " +
			"same paths under your home directory.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	command.AddCommand(skillInstallCmd())
	command.AddCommand(skillUninstallCmd())
	command.AddCommand(skillShowCmd())
	return command
}

func skillInstallCmd() *cobra.Command {
	flags := &skillCmdFlags{}
	command := &cobra.Command{
		Use:   "install",
		Short: "Install the devbox skill for this project and your user",
		Example: "\n  devbox skill install                 # project and user" +
			"\n  devbox skill install --scope project # only this project" +
			"\n  devbox skill install --scope user    # only your user",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillCmd(cmd, flags, skill.Install)
		},
	}
	flags.register(command)
	command.Flags().BoolVarP(
		&flags.force, "force", "f", false,
		"overwrite an existing devbox SKILL.md even if devbox didn't create it",
	)
	return command
}

func skillUninstallCmd() *cobra.Command {
	flags := &skillCmdFlags{}
	command := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the devbox skill from this project and your user",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillCmd(cmd, flags, skill.Uninstall)
		},
	}
	flags.register(command)
	command.Flags().BoolVarP(
		&flags.force, "force", "f", false,
		"remove the devbox SKILL.md even if devbox didn't create it",
	)
	return command
}

func skillShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print the devbox skill",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := io.WriteString(cmd.OutOrStdout(), skill.Content())
			return err
		},
	}
}

func (f *skillCmdFlags) register(cmd *cobra.Command) {
	f.config.register(cmd)
	cmd.Flags().StringVarP(
		&f.scope, "scope", "s", skillScopeAll,
		"where to install: project, user, or all",
	)
}

type skillAction func(scope skill.Scope, root string, force bool) ([]skill.Result, error)

func runSkillCmd(cmd *cobra.Command, flags *skillCmdFlags, action skillAction) error {
	var scopes []skill.Scope
	switch flags.scope {
	case skillScopeProject:
		scopes = []skill.Scope{skill.ScopeProject}
	case skillScopeUser:
		scopes = []skill.Scope{skill.ScopeUser}
	case skillScopeAll:
		scopes = []skill.Scope{skill.ScopeProject, skill.ScopeUser}
	default:
		return fmt.Errorf("invalid --scope %q (must be project, user or all)", flags.scope)
	}

	for _, scope := range scopes {
		root := ""
		if scope == skill.ScopeProject {
			var err error
			root, err = findProjectRoot(flags.config.path)
			if errors.Is(err, devconfig.ErrNotFound) && flags.scope == skillScopeAll {
				ux.Fwarningf(cmd.ErrOrStderr(),
					"No devbox.json found, skipping the project skill. Run `devbox init` to create a project.\n")
				continue
			}
			if err != nil {
				return err
			}
		}
		results, err := action(scope, root, flags.force)
		printSkillResults(cmd.ErrOrStderr(), results)
		if err != nil {
			return err
		}
	}
	return nil
}

func findProjectRoot(path string) (string, error) {
	if path == "" {
		path = "."
	}
	cfg, err := devconfig.Find(path)
	if err != nil {
		return "", err
	}
	return filepath.Dir(cfg.Root.AbsRootPath), nil
}

func printSkillResults(w io.Writer, results []skill.Result) {
	for _, r := range results {
		path := displayPath(r.Path)
		switch r.Status {
		case skill.StatusSkipped:
			ux.Fwarningf(w, "%s was not created by devbox, leaving it alone (use --force to replace it)\n", path)
		case skill.StatusNotFound:
			// Nothing to report.
		default:
			fmt.Fprintf(w, "%s %s\n", r.Status, path)
		}
	}
}

// displayPath shortens path relative to the working directory or home
// directory so output is easier to read.
func displayPath(path string) string {
	if wd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(wd, path); err == nil && filepath.IsLocal(rel) {
			return rel
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		if rel, err := filepath.Rel(home, path); err == nil && filepath.IsLocal(rel) {
			return filepath.Join("~", rel)
		}
	}
	return path
}

// promptInstallSkill asks whether to install the devbox skill after `devbox
// init`. It does nothing when the terminal isn't interactive.
func promptInstallSkill(cmd *cobra.Command, projectDir string) error {
	if envir.IsCI() || !isatty.IsTerminal(os.Stdin.Fd()) || !isatty.IsTerminal(os.Stdout.Fd()) {
		return nil
	}
	const (
		optAll     = "Yes, for this project and my user (recommended)"
		optProject = "Only this project"
		optNo      = "No"
	)
	answer := ""
	err := survey.AskOne(&survey.Select{
		Message: "Install the devbox skill so AI coding agents use devbox in this project?",
		Options: []string{optAll, optProject, optNo},
		Default: optAll,
	}, &answer)
	if err != nil {
		// Treat Ctrl-C or a broken prompt as "no" so init itself still succeeds.
		return nil
	}

	var scopes []skill.Scope
	switch answer {
	case optAll:
		scopes = []skill.Scope{skill.ScopeProject, skill.ScopeUser}
	case optProject:
		scopes = []skill.Scope{skill.ScopeProject}
	default:
		fmt.Fprintln(cmd.ErrOrStderr(), "You can install it later with `devbox skill install`.")
		return nil
	}
	return installSkill(cmd, projectDir, scopes)
}

func installSkill(cmd *cobra.Command, projectDir string, scopes []skill.Scope) error {
	for _, scope := range scopes {
		results, err := skill.Install(scope, projectDir, false)
		printSkillResults(cmd.ErrOrStderr(), results)
		if err != nil {
			return err
		}
	}
	return nil
}
