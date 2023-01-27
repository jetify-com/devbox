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

func (td *TestDevbox) SetDevboxJson(fileContent string) error {
	if err := os.WriteFile("devbox.json", []byte(fileContent), 0666); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (td *TestDevbox) GetDevboxJson() (*impl.Config, error) {
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
	// change into temp directory and run command
	output, err := runCmd(cmd, args)
	// regardless of error or not change back into current working directory
	if err != nil {
		return "", errors.WithStack(err)
	}
	return string(output), nil
}

func Open() *TestDevbox {
	tmpDir, err := os.MkdirTemp(".", ".test_tmp_*")
	if err != nil {
		panic(err)
	}
	os.Chdir(tmpDir)
	return &TestDevbox{
		TmpDir: tmpDir,
	}
}

func (td *TestDevbox) Close() error {
	os.Chdir("..")
	os.Clearenv()
	return os.RemoveAll(td.TmpDir)
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
