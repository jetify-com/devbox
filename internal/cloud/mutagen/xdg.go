package mutagen

// TODO: publish as it's own shared package that other binaries
// can use.

import (
	"os"
	"path/filepath"
)

func DataDir() string {
	return resolveDir("XDG_DATA_HOME", ".local/share")
}

func DataSubpath(subpath string) string {
	return filepath.Join(DataDir(), subpath)
}

func ConfigDir() string {
	return resolveDir("XDG_CONFIG_HOME", ".config")
}

func ConfigSubpath(subpath string) string {
	return filepath.Join(ConfigDir(), subpath)
}

func CacheDir() string {
	return resolveDir("XDG_CACHE_HOME", ".cache")
}

func CacheSubpath(subpath string) string {
	return filepath.Join(CacheDir(), subpath)
}

func StateDir() string {
	return resolveDir("XDG_STATE_HOME", ".local/state")
}

func StateSubpath(subpath string) string {
	return filepath.Join(StateDir(), subpath)
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
