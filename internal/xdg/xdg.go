package xdg

import (
	"os"
	"path/filepath"

	"go.jetpack.io/devbox/internal/env"
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

func dataDir() string   { return resolveDir(env.XDGDataHome, ".local/share") }
func configDir() string { return resolveDir(env.XDGConfigHome, ".config") }
func cacheDir() string  { return resolveDir(env.XDGCacheHome, ".cache") }
func stateDir() string  { return resolveDir(env.XDGStateHome, ".local/state") }

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
