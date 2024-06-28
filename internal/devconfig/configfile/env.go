package configfile

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	file, err := os.Open(c.EnvFrom)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %s", c.EnvFrom)
	}
	defer file.Close()

	envMap := map[string]string{}

	// Read the file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Ideally .env file shouldn't have empty lines and comments but
		// this check makes it allowed.
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line in .env file: %s", line)
		}
		// Also ideally, .env files should not have space in their `key=value` format
		// but this allows `key = value` to pass through as well
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Add the parsed key-value pair to the map
		envMap[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read env file: %v", err)
	}
	return envMap, nil
}
