// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package conf

import (
	"os"
	"strings"
)

// SplitDefault splits a variable reference as passed to an os.Expand mapping
// function into its name and default value. It supports the shell-style
// "NAME:-default" syntax (as in ${NAME:-default}), where default is used if
// NAME is unset or empty. ok is false if the reference has no default.
func SplitDefault(ref string) (name, def string, ok bool) {
	return strings.Cut(ref, ":-")
}

func OSExpandEnvMap(env, existingEnv map[string]string, projectDir string) map[string]string {
	var mapperfunc func(value string) string
	mapperfunc = func(value string) string {
		name, def, hasDefault := SplitDefault(value)

		// Special variables that should return correct value
		var val string
		if name == "PWD" {
			val = projectDir
		} else if existingEnv != nil {
			val = existingEnv[name]
		}

		if val == "" && hasDefault {
			// Allow the default to reference other variables,
			// e.g. ${FOO:-$BAR}.
			return os.Expand(def, mapperfunc)
		}
		return val
	}

	res := map[string]string{}
	for k, v := range env {
		res[k] = os.Expand(v, mapperfunc)
	}
	return res
}
