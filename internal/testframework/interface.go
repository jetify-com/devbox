// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package testframework

import (
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/impl"
	testImpl "go.jetpack.io/devbox/internal/testframework/impl"
)

type TestDevbox interface {
	// Setting up the environment to run a devbox command
	GetTestDir() string
	SetDevboxJson(fileContent string) error
	GetDevboxJson() (*impl.Config, error)

	// Running specific devbox commands and asserting their output
	Close() error
	RunCommand(cmd *cobra.Command, args ...string) (string, error)
	// Add(cmd *cobra.Command, pkgs ...string) (string, error)
	// Generate(subcommand string) (string, error)
	// Info(pkg string, markdown bool) (string, error)
	// Init() (string, error)
	// Rm(pkgs ...string) (string, error)
	// Run(script string) (string, error)
	// Shell() (string, error)
	// Version() (string, error)
	//... and other devbox commands
}

func Open() TestDevbox {
	return testImpl.Open()
}
