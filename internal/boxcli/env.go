// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// to be composed into xyzCmdFlags structs
type envFlag struct {
	env     map[string]string
	envFile string
}

func (f *envFlag) register(cmd *cobra.Command) {
	cmd.PersistentFlags().StringToStringVarP(
		&f.env, "env", "e", nil, "environment variables to set in the devbox environment",
	)
	cmd.PersistentFlags().StringVar(
		&f.envFile, "env-file", "", "path to a file containing environment variables to set in the devbox environment",
	)
}

func (f *envFlag) Env(path string) (map[string]string, error) {
	envs := map[string]string{}
	var err error
	if f.envFile != "" {
		envPath := f.envFile
		if !filepath.IsAbs(envPath) {
			envPath = filepath.Join(path, envPath)
		}
		envs, err = godotenv.Read(envPath)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	for k, v := range f.env {
		envs[k] = v
	}

	return envs, nil
}
