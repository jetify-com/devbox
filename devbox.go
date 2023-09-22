// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

// Package devbox creates isolated development environments.
package devbox

import (
	"context"
	"io"

	"go.jetpack.io/devbox/internal/devconfig"
	"go.jetpack.io/devbox/internal/impl"
	"go.jetpack.io/devbox/internal/impl/devopt"
	"go.jetpack.io/devbox/internal/services"
)

// Devbox provides an isolated development environment.
type Devbox interface {
	// Add adds Nix packages to the config so that they're available in the devbox
	// environment. It validates that the Nix packages exist, and install them.
	// Adding duplicate packages is a no-op.
	Add(ctx context.Context, platforms, excludePlatforms []string, pkgs ...string) error
	Config() *devconfig.Config
	ProjectDir() string
	// Generate creates the directory of Nix files and the Dockerfile that define
	// the devbox environment.
	Generate(ctx context.Context) error
	GenerateDevcontainer(ctx context.Context, generateOpts devopt.GenerateOpts) error
	GenerateDockerfile(ctx context.Context, generateOpts devopt.GenerateOpts) error
	GenerateEnvrcFile(ctx context.Context, force bool, envFlags devopt.EnvFlags) error
	Info(ctx context.Context, pkg string, markdown bool) error
	Install(ctx context.Context) error
	IsEnvEnabled() bool
	ListScripts() []string
	PrintEnv(ctx context.Context, includeHooks bool) (string, error)
	PrintEnvVars(ctx context.Context) ([]string, error)
	PrintGlobalList() error
	Pull(ctx context.Context, opts devopt.PullboxOpts) error
	Push(ctx context.Context, opts devopt.PullboxOpts) error
	// Remove removes Nix packages from the config so that it no longer exists in
	// the devbox environment.
	Remove(ctx context.Context, pkgs ...string) error
	RestartServices(ctx context.Context, services ...string) error
	RunScript(ctx context.Context, scriptName string, scriptArgs []string) error
	Services() (services.Services, error)
	// Shell generates the devbox environment and launches nix-shell as a child process.
	Shell(ctx context.Context) error
	StartProcessManager(ctx context.Context, requestedServices []string, background bool, processComposeFileOrDir string) error
	StartServices(ctx context.Context, services ...string) error
	StopServices(ctx context.Context, allProjects bool, services ...string) error
	ListServices(ctx context.Context) error

	Update(ctx context.Context, opts devopt.UpdateOpts) error
}

// Open opens a devbox by reading the config file in dir.
func Open(opts *devopt.Opts) (Devbox, error) {
	return impl.Open(opts)
}

// InitConfig creates a default devbox config file if one doesn't already exist.
func InitConfig(dir string, writer io.Writer) (bool, error) {
	return devconfig.Init(dir, writer)
}

func GlobalDataPath() (string, error) {
	return impl.GlobalDataPath()
}

func PrintEnvrcContent(w io.Writer, envFlags devopt.EnvFlags) error {
	return impl.PrintEnvrcContent(w, envFlags)
}

// ExportifySystemPathWithoutWrappers reads $PATH, removes `virtenv/.wrappers/bin` paths,
// and returns a string of the form `export PATH=....`
//
// This small utility function could have been inlined in the boxcli caller, but
// needed the impl.exportify functionality. It does not depend on core-devbox.
func ExportifySystemPathWithoutWrappers() string {
	return impl.ExportifySystemPathWithoutWrappers()
}
