package configfile

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-envparse"
)

// IsJetifyCloudEnvFrom reports whether env_from points at Jetify Cloud
// secrets. That feature has been removed, but configs in the wild still set it,
// so we recognize the value in order to ignore it with a warning rather than
// fail on it.
func (c *ConfigFile) IsJetifyCloudEnvFrom() bool {
	// envsec and jetpack-cloud are legacy spellings of jetify-cloud.
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
		// Wrap the underlying error (which is os.ErrNotExist for a missing
		// file) so callers can distinguish a missing env_from file from a
		// genuine parse error via errors.Is(err, os.ErrNotExist).
		return nil, fmt.Errorf("failed to open file: %s: %w", envFileAbsPath, err)
	}
	defer file.Close()

	envMap, err := envparse.Parse(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse env file: %v", err)
	}

	return envMap, nil
}

func (c *ConfigFile) SetEnv(env map[string]string) {
	c.Env = env
	c.ast.setEnv(env)
}
