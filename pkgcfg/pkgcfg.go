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
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	CreateFiles     map[string]string `json:"create_files"`
	Env             map[string]string `json:"env"`
	localConfigPath string            `json:"-"`
}

func CreateFiles(pkg, basePath string) error {
	cfg, err := get(pkg)
	if err != nil {
		return err
	}
	for name, contentPath := range cfg.CreateFiles {
		filePath := filepath.Join(basePath, name)
		content, err := os.ReadFile(filepath.Join(cfg.localConfigPath, contentPath))
		if err != nil {
			return errors.WithStack(err)
		}
		if err = createDir(filepath.Dir(filePath)); err != nil {
			return err
		}
		if err := os.WriteFile(filePath, content, 0744); err != nil {
			return errors.WithStack(err)
		}
		if err := createSymlink(basePath, filePath); err != nil {
			return err
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

func get(pkg string) (*config, error) {
	if configPath := os.Getenv(localPkgConfigPath); configPath != "" {
		debug.Log("Using local package config at %q", configPath)
		return getLocalConfig(configPath, pkg)
	}
	return &config{}, nil
}

func getLocalConfig(configPath, pkg string) (*config, error) {
	pkgConfigPath := filepath.Join(configPath, pkg+".json")
	if _, err := os.Stat(pkgConfigPath); errors.Is(err, os.ErrNotExist) {
		// We don't need config for all packages and that's fine
		return &config{}, nil
	}
	debug.Log("Reading local package config at %q", pkgConfigPath)
	content, err := os.ReadFile(pkgConfigPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	cfg := &config{localConfigPath: configPath}
	if err = json.Unmarshal(content, cfg); err != nil {
		return nil, errors.WithStack(err)
	}
	return cfg, nil
}
func createDir(path string) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func createSymlink(root, filePath string) error {
	name := filepath.Base(filePath)
	newname := filepath.Join(root, ".devbox/conf/bin", name)

	// Create bin path just in case it doesn't exist
	if err := os.MkdirAll(filepath.Join(root, ".devbox/conf/bin"), 0755); err != nil {
		return errors.WithStack(err)
	}

	if _, err := os.Stat(newname); err == nil {
		if err = os.Remove(newname); err != nil {
			return errors.WithStack(err)
		}
	}

	if err := os.Symlink(filePath, newname); err != nil {
		return errors.WithStack(err)
	}
	return nil
}
