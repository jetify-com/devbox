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
	"go.jetpack.io/devbox/internal/impl"
)

type TestDevbox struct {
	TmpDir string
}

func (td *TestDevbox) SetEnv(key string, value string) error {
	return os.Setenv(key, value)
}

func (td *TestDevbox) GetTestDir() string {
	return td.TmpDir
}

func (td *TestDevbox) SetDevboxJSON(fileContent string) error {
	if err := os.WriteFile("devbox.json", []byte(fileContent), 0666); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (td *TestDevbox) GetDevboxJSON() (*impl.Config, error) {
	file, err := os.ReadFile("devbox.json")
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

func (td *TestDevbox) CreateFile(fileName string, fileContent string) error {
	if err := os.WriteFile(fileName, []byte(fileContent), 0666); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (td *TestDevbox) RunCommand(cmd *cobra.Command, args ...string) (string, error) {
	b := bytes.NewBufferString("")
	cmd.SetErr(b)
	cmd.SetOut(b)
	cmd.SetArgs(args)
	// execute command
	err := cmd.Execute()
	if err != nil {
		return "", errors.WithStack(err)
	}
	// read command output
	out, err := io.ReadAll(b)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return string(out), nil
}

func Open() *TestDevbox {
	tmpDir, err := os.MkdirTemp(".", ".test_tmp_*")
	if err != nil {
		panic(err)
	}
	err = os.Chdir(tmpDir)
	if err != nil {
		panic(err)
	}
	return &TestDevbox{
		TmpDir: tmpDir,
	}
}

func (td *TestDevbox) Close() error {
	err := os.Chdir("..")
	if err != nil {
		return errors.WithMessage(err, "failed to change directory")
	}
	os.Clearenv()
	err = os.RemoveAll(td.TmpDir)
	if err != nil {
		return errors.WithMessagef(err, "failed to delete directory: %s", td.TmpDir)
	}
	return nil
}
