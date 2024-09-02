package testrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"

	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/envir"
	"go.jetpack.io/devbox/internal/xdg"
)

// setupTestEnv configures env for devbox tests.
func setupTestEnv(env *testscript.Env) error {
	setupPATH(env)
	setupHome(env)
	setupCacheHome(env)
	propagateEnvVars(env,
		debug.DevboxDebug, // to enable extra logging
		"SSL_CERT_FILE",   // so HTTPS works with Nix-installed certs
	)
	return nil
}

// setupHome sets the test's HOME to a unique temp directory. The testscript
// package sets it to /no-home by default (presumably to improve isolation), but
// this breaks most programs.
func setupHome(env *testscript.Env) {
	env.Setenv(envir.Home, env.T().(testing.TB).TempDir())
}

// setupPATH removes all directories from the test's PATH to ensure that it only
// uses the PATH set by devbox. The one exception is the testscript's bin
// directory, which contains the commands given to testscript.RunMain
// (such as devbox itself).
func setupPATH(env *testscript.Env) {
	s, _, _ := strings.Cut(env.Getenv(envir.Path), string(filepath.ListSeparator))
	env.Setenv(envir.Path, s)
}

// setupCacheHome sets the test's XDG_CACHE_HOME to a unique temp directory so
// that it doesn't share caches with other tests or the user's system. For
// programs where this would make tests too slow, it symlinks specific cache
// subdirectories to a shared location that persists between test runs. For
// example, $WORK/.cache/nix would symlink to $XDG_CACHE_HOME/devbox-tests/nix
// so that Nix doesn't re-download tarballs for every test.
func setupCacheHome(env *testscript.Env) {
	t := env.T().(testing.TB) //nolint:varnamelen

	cacheHome := filepath.Join(env.WorkDir, ".cache")
	env.Setenv(envir.XDGCacheHome, cacheHome)
	err := os.MkdirAll(cacheHome, 0o755)
	if err != nil {
		t.Fatal("create XDG_CACHE_HOME for test:", err)
	}

	// Symlink cache subdirectories that we want to share and persist
	// between tests.
	sharedCacheDir := xdg.CacheSubpath("devbox-tests")
	for _, subdir := range []string{"nix", "pip"} {
		sharedSubdir := filepath.Join(sharedCacheDir, subdir)
		err := os.MkdirAll(sharedSubdir, 0o755)
		if err != nil {
			t.Fatal("create shared XDG_CACHE_HOME subdir:", err)
		}

		testSubdir := filepath.Join(cacheHome, subdir)
		err = os.Symlink(sharedSubdir, testSubdir)
		if err != nil {
			t.Fatal("symlink test's XDG_CACHE_HOME subdir to shared XDG_CACHE_HOME subdir:", err)
		}
	}
}

// propagateEnvVars propagates the values of environment variables to the test
// environment.
func propagateEnvVars(env *testscript.Env, vars ...string) {
	for _, key := range vars {
		if v, ok := os.LookupEnv(key); ok {
			env.Setenv(key, v)
		}
	}
}
