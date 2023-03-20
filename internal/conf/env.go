package conf

import (
	"os"
)

func OSExpandEnvMap(
	env map[string]string,
	projectDir string,
	existingEnv map[string]string,
) map[string]string {
	mapperfunc := func(value string) string {
		// Special variables that should return correct value
		switch value {
		case "PWD":
			return projectDir
		}
		// check if referenced variables exists in computed environment
		if v, ok := existingEnv[value]; ok {
			return v
		}
		return ""
	}

	res := map[string]string{}
	for k, v := range env {
		res[k] = os.Expand(v, mapperfunc)
	}
	return res
}
