package plugin

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/plugins"
)

func getConfigIfAny(pkg, projectDir string) (*Config, error) {
	configFiles, err := plugins.BuiltIn.ReadDir(".")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Try to find perfect match first
	for _, file := range configFiles {
		if file.IsDir() || strings.HasSuffix(file.Name(), ".go") {
			continue
		}
		content, err := plugins.BuiltIn.ReadFile(file.Name())
		if err != nil {
			return nil, errors.WithStack(err)
		}

		cfg, err := buildConfig(pkg, projectDir, string(content))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		// if match regex is set we use it to check. Otherwise we assume it's a
		// perfect match
		if (cfg.Match != "" && !regexp.MustCompile(cfg.Match).MatchString(pkg)) ||
			(cfg.Match == "" && strings.Split(file.Name(), ".")[0] != pkg) {
			continue
		}
		return cfg, nil
	}
	return nil, nil
}

func getFileContent(contentPath string) ([]byte, error) {
	return plugins.BuiltIn.ReadFile(contentPath)
}
