// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

// Package devbox creates isolated development environments.
package devbox

import (
	"context"
	"io"

	"go.jetpack.io/devbox/internal/impl"
	"go.jetpack.io/devbox/internal/planner/plansdk"
	"go.jetpack.io/devbox/internal/plugin"
)

// Devbox provides an isolated development environment.
type Devbox interface {
	// Add adds a Nix package to the config so that it's available in the devbox
	// environment. It validates that the Nix package exists, but doesn't install
	// it. Adding a duplicate package is a no-op.
	Add(pkgs ...string) error
	AddGlobal(pkgs ...string) error
	Config() *impl.Config
	ProjectDir() string
	Exec(cmds ...string) error
	// Generate creates the directory of Nix files and the Dockerfile that define
	// the devbox environment.
	Generate() error
	GenerateDevcontainer(force bool) error
	GenerateDockerfile(force bool) error
	GenerateEnvrc(force bool, source string) error
	Info(pkg string, markdown bool) error
	ListScripts() []string
	PrintEnv() (string, error)
	PrintGlobalList() error
	// Remove removes Nix packages from the config so that it no longer exists in
	// the devbox environment.
	Remove(pkgs ...string) error
	RemoveGlobal(pkgs ...string) error
	RunScript(scriptName string, scriptArgs []string) error
	// TODO: Deprecate in favor of RunScript
	RunScriptInShell(scriptName string) error
	Services() (plugin.Services, error)
	// Shell generates the devbox environment and launches nix-shell as a child
	// process.
	Shell() error
	// ShellPlan creates a plan of the actions that devbox will take to generate its
	// shell environment.
	ShellPlan() (*plansdk.ShellPlan, error)
	StartServices(ctx context.Context, services ...string) error
	StopServices(ctx context.Context, services ...string) error
}

// Open opens a devbox by reading the config file in dir.
func Open(dir string, writer io.Writer) (Devbox, error) {
	return impl.Open(dir, writer)
}

// InitConfig creates a default devbox config file if one doesn't already
// exist.
func InitConfig(dir string, writer io.Writer) (bool, error) {
	return impl.InitConfig(dir, writer)
}

func IsDevboxShellEnabled() bool {
	return impl.IsDevboxShellEnabled()
}

func GlobalConfigPath() (string, error) {
	return impl.GlobalConfigPath()
}
