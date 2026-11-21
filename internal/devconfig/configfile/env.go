package configfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/hashicorp/go-envparse"
)

// EnvFrom lists the sources to load env variables from, in order. In
// devbox.json it is either a single string or an array of strings.
type EnvFrom []string

func (e *EnvFrom) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*e = nil
		if single != "" {
			*e = EnvFrom{single}
		}
		return nil
	}
	return json.Unmarshal(data, (*[]string)(e))
}

func (e EnvFrom) MarshalJSON() ([]byte, error) {
	if len(e) == 1 {
		return json.Marshal(e[0])
	}
	return json.Marshal([]string(e))
}

// IsJetifyCloudEnvFrom reports whether env_from points at Jetify Cloud
// secrets. That feature has been removed, but configs in the wild still set it,
// so we recognize the value in order to ignore it with a warning rather than
// fail on it.
func (c *ConfigFile) IsJetifyCloudEnvFrom() bool {
	return slices.ContainsFunc(c.EnvFrom, IsJetifyCloudSource)
}

func IsJetifyCloudSource(source string) bool {
	// envsec and jetpack-cloud are legacy spellings of jetify-cloud.
	return source == "envsec" || source == "jetpack-cloud" || source == "jetify-cloud"
}

// ExecCommand returns the command of an "exec:<command>" source.
func ExecCommand(source string) (string, bool) {
	command, ok := strings.CutPrefix(source, "exec:")
	if !ok {
		return "", false
	}
	return strings.TrimSpace(command), true
}

func IsDotEnvSource(source string) bool {
	_, isExec := ExecCommand(source)
	return !isExec && filepath.Ext(source) == ".env"
}

func (c *ConfigFile) ParseEnvsFromDotEnv(source string) (map[string]string, error) {
	if !IsDotEnvSource(source) {
		return nil, fmt.Errorf("env file does not have a .env extension")
	}
	envFileAbsPath := source
	if !filepath.IsAbs(source) {
		envFileAbsPath = filepath.Join(filepath.Dir(c.AbsRootPath), source)
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
