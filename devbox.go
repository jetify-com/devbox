// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

// Package devbox creates isolated development environments.
package devbox

import (
	"context"
	"io"

	"go.jetpack.io/devbox/internal/devconfig"
	"go.jetpack.io/devbox/internal/impl"
	"go.jetpack.io/devbox/internal/planner/plansdk"
	"go.jetpack.io/devbox/internal/services"
)

// Devbox provides an isolated development environment.
type Devbox interface {
	// Add adds Nix packages to the config so that they're available in the devbox
	// environment. It validates that the Nix packages exist, and install them.
	// Adding duplicate packages is a no-op.
	Add(ctx context.Context, pkgs ...string) error
	Config() *devconfig.Config
	ProjectDir() string
	// Generate creates the directory of Nix files and the Dockerfile that define
	// the devbox environment.
	Generate() error
	GenerateDevcontainer(force bool) error
	GenerateDockerfile(force bool) error
	GenerateEnvrcFile(force bool) error
	Info(pkg string, markdown bool) error
	Install(ctx context.Context) error
	IsEnvEnabled() bool
	ListScripts() []string
	PrintEnv(ctx context.Context, includeHooks bool) (string, error)
	PrintGlobalList() error
	Pull(ctx context.Context, overwrite bool, path string) error
	Push(url string) error
	// Remove removes Nix packages from the config so that it no longer exists in
	// the devbox environment.
	Remove(ctx context.Context, pkgs ...string) error
	RestartServices(ctx context.Context, services ...string) error
	RunScript(ctx context.Context, scriptName string, scriptArgs []string) error
	Services() (services.Services, error)
	// Shell generates the devbox environment and launches nix-shell as a child process.
	Shell(ctx context.Context) error
	// ShellPlan creates a plan of the actions that devbox will take to generate its
	// shell environment.
	ShellPlan() (*plansdk.FlakePlan, error)
	StartProcessManager(ctx context.Context, requestedServices []string, background bool, processComposeFileOrDir string) error
	StartServices(ctx context.Context, services ...string) error
	StopServices(ctx context.Context, allProjects bool, services ...string) error
	ListServices(ctx context.Context) error

	Update(ctx context.Context, pkgs ...string) error
}

// Open opens a devbox by reading the config file in dir.
func Open(dir string, writer io.Writer) (Devbox, error) {
	return impl.Open(dir, writer, true)
}

func OpenWithoutWarnings(dir string, writer io.Writer) (Devbox, error) {
	return impl.Open(dir, writer, false)
}

// InitConfig creates a default devbox config file if one doesn't already exist.
func InitConfig(dir string, writer io.Writer) (bool, error) {
	return devconfig.Init(dir, writer)
}

func GlobalDataPath() (string, error) {
	return impl.GlobalDataPath()
}

func PrintEnvrcContent(w io.Writer) error {
	return impl.PrintEnvrcContent(w)
}
