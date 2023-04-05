package xdg

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func DataSubpath(subpath string) string {
	return filepath.Join(dataDir(), subpath)
}

func ConfigSubpath(subpath string) string {
	return filepath.Join(configDir(), subpath)
}

func CacheSubpath(subpath string) string {
	return filepath.Join(cacheDir(), subpath)
}

func StateSubpath(subpath string) string {
	return filepath.Join(stateDir(), subpath)
}

func RuntimeSubpath(subpath string) (string, error) {
	dir, err := runtimeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, subpath), nil
}

func dataDir() string   { return resolveDir("XDG_DATA_HOME", ".local/share") }
func configDir() string { return resolveDir("XDG_CONFIG_HOME", ".config") }
func cacheDir() string  { return resolveDir("XDG_CACHE_HOME", ".cache") }
func stateDir() string  { return resolveDir("XDG_STATE_HOME", ".local/state") }

func runtimeDir() (string, error) {
	dir := resolveDir("XDG_RUNTIME_DIR", ".local/run")
	// Ensure the directory exists with correct permissions, as per XDG spec
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", errors.WithStack(err)
	}
	return dir, nil
}

func resolveDir(envvar string, defaultPath string) string {
	dir := os.Getenv(envvar)
	if dir != "" {
		return dir
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = "~"
	}

	return filepath.Join(home, defaultPath)
}
