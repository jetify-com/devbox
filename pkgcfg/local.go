package pkgcfg

import (
	"net/url"
	"os"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/debug"
)

func getLocalConfig(configPath, pkg, rootDir string) (*config, error) {
	pkgConfigPath, err := getLocalBestConfigPath(configPath, pkg)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if pkgConfigPath == "" {
		return &config{}, nil
	}

	debug.Log("Reading local package config at %q", pkgConfigPath)
	content, err := os.ReadFile(pkgConfigPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	cfg := &config{localConfigPath: configPath}
	return buildConfig(cfg, pkg, rootDir, string(content))
}

func getLocalBestConfigPath(configPath, pkg string) (string, error) {
	files, err := os.ReadDir(configPath)
	if err != nil {
		return "", errors.WithStack(err)
	}

	// Try to find perfect match first
	for _, file := range files {
		if file.Name() == pkg+".json" {
			return url.JoinPath(configPath, file.Name())
		}
	}
	for _, file := range files {
		if wildcardMatch(file.Name(), pkg+".json") {
			return url.JoinPath(configPath, file.Name())
		}
	}
	return "", nil
}
