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
	SetEnv(key string, value string) error
	SetDevboxJSON(fileContent string) error
	GetDevboxJSON() (*impl.Config, error)
	CreateFile(fileName string, fileContent string) error
	Close() error

	// Running specific devbox commands and asserting their output
	RunCommand(cmd *cobra.Command, args ...string) (string, error)
}

func Open() TestDevbox {
	return testImpl.Open()
}
