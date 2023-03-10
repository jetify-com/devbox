package testrunner

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/rogpeppe/go-internal/testscript"
	"go.jetpack.io/devbox/internal/xdg"
)

func setupTestEnv(env *testscript.Env) error {
	setupPATH(env)

	if err := setupXDGHomes(env); err != nil {
		return err
	}

	err := setupSharedCacheDirectories(env)
	if err != nil {
		return err
	}

	// Enable new `devbox run` so we can use it in tests. This is temporary,
	// and should be removed once we enable this feature flag.
	env.Setenv("DEVBOX_FEATURE_UNIFIED_ENV", "1")
	return nil
}

func setupPATH(env *testscript.Env) {
	// Ensure path is empty so that we rely only on the PATH set by devbox
	// itself.
	// The one entry we need to keep is the /bin directory in the testing directory.
	// That directory is setup by the testing framework itself, and it's what allows
	// us to call our own custom "devbox" command.
	oldPath := env.Getenv("PATH")
	newPath := strings.Split(oldPath, ":")[0]
	env.Setenv("PATH", newPath)
}

func setupSharedCacheDirectories(env *testscript.Env) error {
	// There is one directory we do want to share across tests: nix's cache.
	// Without it tests are very slow, and nix would end up re-downloading
	// nixpkgs every time.
	// Here we create a shared location for nix's cache, and symlink from
	// the test's working directory.
	err := os.MkdirAll(xdg.CacheSubpath("devbox-tests/nix"), 0755) // Ensure dir exists.
	if err != nil {
		return err
	}

	cacheHome := xdgHomePath(env, "XDG_CACHE_HOME")
	err = os.Symlink(xdg.CacheSubpath("devbox-tests/nix"), filepath.Join(cacheHome, "nix"))
	if err != nil {
		return err
	}

	return nil
}

var xdgHomes = map[string]string{
	"XDG_CACHE_HOME":  ".cache",
	"XDG_CONFIG_HOME": ".config",
	"XDG_DATA_HOME":   ".share",
	"XDG_STATE_HOME":  ".state",
}

func setupXDGHomes(env *testscript.Env) error {
	for envKey := range xdgHomes {
		if err := setupXDGHome(env, envKey); err != nil {
			return err
		}
	}
	return nil
}

// setupXDGHome enables testscripts to use particular XDG directories.
//
// Both devbox itself and nix occasionally create some files in
// XDG folders. For testscripts, set the XDG folders
// to a location within the test's working directory.
func setupXDGHome(env *testscript.Env, envKey string) error {
	path := xdgHomePath(env, envKey)
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	env.Setenv(envKey, path)
	return nil
}

func xdgHomePath(env *testscript.Env, envKey string) string {
	return filepath.Join(env.WorkDir, xdgHomes[envKey])
}
