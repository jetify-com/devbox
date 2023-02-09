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

	err := setupCacheHome(env)
	if err != nil {
		return err
	}

	// Enable new `devbox run` so we can use it in tests. This is temporary,
	// and should be removed once we enable this feature flag.
	env.Setenv("DEVBOX_FEATURE_STRICT_RUN", "1")
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

func setupCacheHome(env *testscript.Env) error {
	// Both devbox itself and nix occasionally create some files in
	// XDG_CACHE_HOME (which defaults to ~/.cache). For purposes of this
	// test set it to a location within the test's working directory:
	cacheHome := filepath.Join(env.WorkDir, ".cache")
	env.Setenv("XDG_CACHE_HOME", cacheHome)
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
