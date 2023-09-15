package devconfig

import (
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/integrations/envsec"
)

func (c *Config) ComputedEnv(projectDir string) (map[string]string, error) {
	env := map[string]string{}
	var err error
	if featureflag.Envsec.Enabled() {
		if c.EnvFrom == "envsec" {
			env, err = envsec.Env(projectDir)
			if err != nil {
				return nil, err
			}
		} else if c.EnvFrom != "" {
			return nil, usererr.New("unknown from_env value: %s", c.EnvFrom)
		}
	}
	for k, v := range c.Env {
		env[k] = v
	}
	return env, nil
}
