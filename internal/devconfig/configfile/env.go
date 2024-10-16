package configfile

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-envparse"
)

func (c *ConfigFile) IsEnvsecEnabled() bool {
	// envsec for legacy. jetpack-cloud for legacy
	return c.EnvFrom == "envsec" || c.EnvFrom == "jetpack-cloud" || c.EnvFrom == "jetify-cloud"
}

func (c *ConfigFile) IsdotEnvEnabled() bool {
	// filename has to end with .env
	return filepath.Ext(c.EnvFrom) == ".env"
}

func (c *ConfigFile) ParseEnvsFromDotEnv() (map[string]string, error) {
	// This check should never happen because we call IsdotEnvEnabled
	// before calling this method. But having it makes it more robust
	// in case if anyone uses this method without the IsdotEnvEnabled
	if !c.IsdotEnvEnabled() {
		return nil, fmt.Errorf("env file does not have a .env extension")
	}
	envFileAbsPath := c.EnvFrom
	if !filepath.IsAbs(c.EnvFrom) {
		envFileAbsPath = filepath.Join(filepath.Dir(c.AbsRootPath), c.EnvFrom)
	}
	file, err := os.Open(envFileAbsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %s", envFileAbsPath)
	}
	defer file.Close()

	envMap, err := envparse.Parse(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse env file: %v", err)
	}

	return envMap, nil
}
