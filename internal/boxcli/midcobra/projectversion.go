// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package midcobra

import (
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"go.jetify.com/devbox/internal/devconfig"
	"go.jetify.com/devbox/internal/vercheck"
)

// ProjectVersion returns a middleware that switches to the devbox version
// required by the project's devbox_version before the command runs, when its
// policy is "auto". Opening the project enforces devbox_version too, but only
// after the command's other setup (such as checking the nix install), so
// switching from there pays for devbox startup twice.
//
// The middleware only handles the common case: a command with a --config flag
// that opens a single project. Commands it skips, such as devbox global and
// --all-projects, still switch versions when they open a project.
func ProjectVersion() Middleware {
	return &projectVersionMiddleware{}
}

type projectVersionMiddleware struct{}

var _ Middleware = (*projectVersionMiddleware)(nil)

func (m *projectVersionMiddleware) preRun(cmd *cobra.Command, args []string) {
	configPath, ok := projectConfigPath(cmd, args)
	if !ok {
		return
	}

	var cfg *devconfig.Config
	var err error
	if configPath == "" {
		cfg, err = devconfig.Find(".")
	} else {
		cfg, err = devconfig.Open(configPath)
	}
	if err != nil {
		// Let the command report the error when it opens the project.
		return
	}
	vercheck.SwitchProjectVersion(cfg.Root.AbsRootPath, cfg.Root.DevboxVersion)
}

func (m *projectVersionMiddleware) postRun(*cobra.Command, []string, error) {}

// projectConfigPath returns the value of the --config flag of the subcommand
// that args run. It returns false if the subcommand doesn't open a single
// project from its --config flag.
func projectConfigPath(root *cobra.Command, args []string) (path string, ok bool) {
	subcmd, flagArgs, err := root.Find(args)
	if err != nil {
		return "", false
	}
	// devbox global commands set --config to the global project in a
	// PersistentPreRunE, so the flag's value isn't known yet.
	for c := subcmd; c != nil; c = c.Parent() {
		if c.Name() == "global" && c.Parent() == root {
			return "", false
		}
	}
	configFlag := subcmd.Flag("config")
	if configFlag == nil || configFlag.Value.Type() != "string" {
		return "", false
	}

	// Parse only the flags we need into a separate flag set so that the
	// command's own flags are still unset when cobra parses them.
	flags := pflag.NewFlagSet(subcmd.Name(), pflag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.ParseErrorsAllowlist.UnknownFlags = true
	flags.StringP("config", configFlag.Shorthand, configFlag.DefValue, "")
	allProjects := false
	if subcmd.Flag("all-projects") != nil {
		flags.BoolVar(&allProjects, "all-projects", false, "")
	}
	if err := flags.Parse(flagArgs); err != nil || allProjects {
		return "", false
	}
	path, err = flags.GetString("config")
	return path, err == nil
}
