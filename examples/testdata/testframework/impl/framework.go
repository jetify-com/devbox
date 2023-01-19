// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package impl

import (
	"encoding/json"
	"io/ioutil"
	"os/exec"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/impl/shellcmd"
)

type DevboxJson struct {
	Packages []string `cue:"[...string]" json:"packages"`
	Shell    struct {
		InitHook shellcmd.Commands             `json:"init_hook,omitempty"`
		Scripts  map[string]*shellcmd.Commands `json:"scripts,omitempty"`
	} `json:"shell,omitempty"`

	Nixpkgs struct {
		Commit string `json:"commit,omitempty"`
	} `json:"nixpkgs,omitempty"`
}

type TestDevbox struct {
	devboxJsonPath string
}

func (td *TestDevbox) Info(pkg string, markdown bool) (string, error) {
	cmd := exec.Command("devbox", "info", pkg)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.WithStack(err)
	}
	return string(output), nil
}

func (td *TestDevbox) Version() (string, error) {
	cmd := exec.Command("devbox", "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.WithStack(err)
	}
	return string(output), nil
}

func (td *TestDevbox) Add(pkgs ...string) (string, error) {
	args := append([]string{"add"}, pkgs...)
	cmd := exec.Command("devbox", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.WithStack(err)
	}
	return string(output), nil
}

func (td *TestDevbox) Rm(pkgs ...string) (string, error) {
	args := append([]string{"rm"}, pkgs...)
	cmd := exec.Command("devbox", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.WithStack(err)
	}
	return string(output), nil
}

func (td *TestDevbox) SetDevboxJson(path string) error {
	td.devboxJsonPath = path
	return nil
}

func (td *TestDevbox) GetDevboxJson() (*DevboxJson, error) {
	file, err := ioutil.ReadFile(td.devboxJsonPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	data := &DevboxJson{}
	err = json.Unmarshal(file, &data)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return data, nil
}

func Open() *TestDevbox {
	return &TestDevbox{}
}
