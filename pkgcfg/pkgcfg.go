package pkgcfg

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/debug"
)

const localPkgConfigPath = "DEVBOX_LOCAL_PKG_CONFIG"

type config struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	CreateFiles map[string]string `json:"create_files"`
	Env         map[string]string `json:"env"`
}

func get(pkg string) (*config, error) {
	if configPath := os.Getenv(localPkgConfigPath); configPath != "" {
		debug.Log("Using local package config at %q", configPath)
		return getLocalConfig(configPath, pkg)
	}
	return &config{}, nil
}

func getLocalConfig(configPath, pkg string) (*config, error) {
	configPath = filepath.Join(configPath, pkg+".json")
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		// We don't need config for all packages and that's fine
		return &config{}, nil
	}
	debug.Log("Reading local package config at %q", configPath)
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	cfg := &config{}
	if err = json.Unmarshal(content, cfg); err != nil {
		return nil, errors.WithStack(err)
	}
	return cfg, nil
}

func CreateFiles(pkg, basePath string) error {
	cfg, err := get(pkg)
	if err != nil {
		return err
	}
	for name, contentPath := range cfg.CreateFiles {
		filePath := filepath.Join(basePath, name)
		if _, err := os.Stat(filePath); err == nil {
			continue
		}
		content, err := os.ReadFile(filepath.Join(basePath, contentPath))
		if err != nil {
			return errors.WithStack(err)
		}
		if err := os.WriteFile(filePath, content, 0744); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func Env(pkgs []string) (map[string]string, error) {
	env := map[string]string{}
	for _, pkg := range pkgs {
		cfg, err := get(pkg)
		if err != nil {
			return nil, err
		}
		for k, v := range cfg.Env {
			env[k] = v
		}
	}
	return env, nil
}
