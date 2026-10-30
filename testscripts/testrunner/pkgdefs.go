package testrunner

import (
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
	"github.com/stretchr/testify/require"
)

// RunPackageDefinitionTestscripts runs the testscripts that live next to
// the package definitions in pkgsDir (pkgs/<name>/<name>.test.txt). Each
// script can reach the definitions through $PKGS, and runs with package
// definitions enabled.
func RunPackageDefinitionTestscripts(t *testing.T, pkgsDir string) {
	pkgsDir, err := filepath.Abs(pkgsDir)
	require.NoError(t, err)
	scripts := globScripts(filepath.Join(pkgsDir, "*", "*.test.txt"))
	require.NotEmpty(t, scripts, "no package definition test scripts found")

	params := getTestscriptParams(pkgsDir)
	params.Dir = ""
	params.Files = scripts
	params.Setup = func(env *testscript.Env) error {
		if err := setupTestEnv(env); err != nil {
			return err
		}
		env.Setenv("PKGS", pkgsDir)
		env.Setenv("DEVBOX_FEATURE_PKG_DEFS", "1")
		// Version lookups use the GitHub API, whose anonymous rate limit is
		// easy to hit when every package resolves a few versions.
		propagateEnvVars(env, "GITHUB_TOKEN", "DEVBOX_GITHUB_API_TOKEN")
		return nil
	}
	testscript.Run(t, params)
}
