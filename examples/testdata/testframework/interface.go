// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package testframework

import (
	"go.jetpack.io/devbox/examples/testdata/testframework/impl"
	internalImpl "go.jetpack.io/devbox/internal/impl"
)

type TestDevbox interface {
	// Setting up the environment to run a devbox command
	SetDevboxJson(path string) error
	GetDevboxJson() (*internalImpl.Config, error)

	// Running specific devbox commands and asserting their output
	Add(pkgs ...string) (string, error)
	Generate(subcommand string) (string, error)
	Info(pkg string, markdown bool) (string, error)
	Init() (string, error)
	Rm(pkgs ...string) (string, error)
	Run(script string) (string, error)
	Shell() (string, error)
	Version() (string, error)
	//... and other devbox commands
}

func Open() TestDevbox {
	return impl.Open()
}
