// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package conf

import (
	"os"
)

func OSExpandEnvMap(env, existingEnv map[string]string, projectDir string) map[string]string {
	mapperfunc := func(value string) string {
		// Special variables that should return correct value
		switch value {
		case "PWD":
			return projectDir
		}

		// in case existingEnv is nil
		if existingEnv == nil {
			return ""
		}
		return existingEnv[value]
	}

	res := map[string]string{}
	for k, v := range env {
		res[k] = os.Expand(v, mapperfunc)
	}
	return res
}
