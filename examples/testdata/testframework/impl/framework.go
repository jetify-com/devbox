// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package impl

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/boxcli"
	"go.jetpack.io/devbox/internal/impl"
)

type TestDevbox struct {
	devboxJsonPath string
}

func (td *TestDevbox) SetDevboxJson(path string) error {
	td.devboxJsonPath = path
	return nil
}

func (td *TestDevbox) GetDevboxJson() (*impl.Config, error) {
	file, err := os.ReadFile(td.devboxJsonPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	data := &impl.Config{}
	err = json.Unmarshal(file, &data)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return data, nil
}

func (td *TestDevbox) Add(pkgs ...string) (string, error) {
	cmd := boxcli.AddCmd()
	output, err := runCmd(cmd, pkgs)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return string(output), nil
}

func (td *TestDevbox) Generate(subcommand string) (string, error) {
	cmd := boxcli.GenerateCmd()
	output, err := runCmd(cmd, []string{subcommand})
	if err != nil {
		return "", errors.WithStack(err)
	}
	return string(output), nil
}

func (td *TestDevbox) Info(pkg string, markdown bool) (string, error) {
	cmd := boxcli.InfoCmd()
	output, err := runCmd(cmd, []string{pkg})
	if err != nil {
		return "", errors.WithStack(err)
	}
	return string(output), nil
}

func (td *TestDevbox) Init() (string, error) {
	cmd := boxcli.InitCmd()
	output, err := runCmd(cmd, nil)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return string(output), nil
}

func (td *TestDevbox) Rm(pkgs ...string) (string, error) {
	cmd := boxcli.RemoveCmd()
	output, err := runCmd(cmd, pkgs)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return string(output), nil
}

func (td *TestDevbox) Run(script string) (string, error) {
	cmd := boxcli.RunCmd()
	output, err := runCmd(cmd, []string{script})
	if err != nil {
		return "", errors.WithStack(err)
	}
	return string(output), nil
}

func (td *TestDevbox) Shell() (string, error) {
	cmd := boxcli.ShellCmd()
	output, err := runCmd(cmd, nil)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return string(output), nil
}

func (td *TestDevbox) Version() (string, error) {
	cmd := boxcli.VersionCmd()
	output, err := runCmd(cmd, nil)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return string(output), nil
}

func Open() *TestDevbox {
	return &TestDevbox{}
}

func runCmd(cmd *cobra.Command, args []string) (string, error) {
	b := bytes.NewBufferString("")
	cmd.SetErr(b)
	cmd.SetOut(b)
	cmd.SetArgs(args)
	cmd.Execute()
	out, err := io.ReadAll(b)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
