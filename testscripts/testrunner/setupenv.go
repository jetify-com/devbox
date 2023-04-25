package testrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
	"go.jetpack.io/devbox/internal/env"
	"go.jetpack.io/devbox/internal/xdg"
)

func setupTestEnv(t *testing.T, envs *testscript.Env) error {
	setupPATH(envs)

	setupHome(t, envs)

	err := setupCacheHome(envs)
	if err != nil {
		return err
	}

	envs.Setenv(env.DevboxDebug, os.Getenv(env.DevboxDebug))
	return nil
}

func setupHome(t *testing.T, envs *testscript.Env) {
	// We set a HOME env-var because:
	// 1. testscripts overrides it to /no-home, presumably to improve isolation
	// 2. but many language tools rely on a $HOME being set, and break due to 1.
	//    examples include ~/.dotnet folder and GOCACHE=$HOME/Library/Caches/go-build
	envs.Setenv(env.Home, t.TempDir())
}

func setupPATH(envs *testscript.Env) {
	// Ensure path is empty so that we rely only on the PATH set by devbox
	// itself.
	// The one entry we need to keep is the /bin directory in the testing directory.
	// That directory is setup by the testing framework itself, and it's what allows
	// us to call our own custom "devbox" command.
	oldPath := envs.Getenv(env.Path)
	newPath := strings.Split(oldPath, ":")[0]
	envs.Setenv(env.Path, newPath)
}

func setupCacheHome(envs *testscript.Env) error {
	// Both devbox itself and nix occasionally create some files in
	// XDG_CACHE_HOME (which defaults to ~/.cache). For purposes of this
	// test set it to a location within the test's working directory:
	cacheHome := filepath.Join(envs.WorkDir, ".cache")
	envs.Setenv(env.XDGCacheHome, cacheHome)
	err := os.MkdirAll(cacheHome, 0755) // Ensure dir exists.
	if err != nil {
		return err
	}

	// There is one directory we do want to share across tests: nix's cache.
	// Without it tests are very slow, and nix would end up re-downloading
	// nixpkgs every time.
	// Here we create a shared location for nix's cache, and symlink from
	// the test's working directory.
	err = os.MkdirAll(xdg.CacheSubpath("devbox-tests/nix"), 0755) // Ensure dir exists.
	if err != nil {
		return err
	}
	err = os.Symlink(xdg.CacheSubpath("devbox-tests/nix"), filepath.Join(cacheHome, "nix"))
	if err != nil {
		return err
	}

	return nil
}
