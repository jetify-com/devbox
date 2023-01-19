// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package testframework

import "go.jetpack.io/devbox/examples/testdata/testframework/impl"

type TestDevbox interface {
	// Setting up the environment to run a devbox command
	SetDevboxJson(path string) error
	GetDevboxJson() (*impl.DevboxJson, error)
	// SetEnvVariables(input map[string]string) error
	// SetConfigFile(path string, file []byte) error

	// Running specific devbox commands and asserting their output
	Add(pkgs ...string) (string, error)
	Rm(pkgs ...string) (string, error)
	// Shell() (string, error)
	// Run(script string) (string, error)
	Info(pkg string, markdown bool) (string, error)
	Version() (string, error)
	//... and other devbox commands
}

func Open() TestDevbox {
	return impl.Open()
}
