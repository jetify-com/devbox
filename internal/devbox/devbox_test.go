// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.jetify.com/devbox/internal/devbox/envpath"

	"go.jetify.com/devbox/internal/devbox/devopt"
	"go.jetify.com/devbox/internal/devconfig"
	"go.jetify.com/devbox/internal/envir"
	"go.jetify.com/devbox/internal/nix"
)

func TestDevbox(t *testing.T) {
	t.Setenv("TMPDIR", "/tmp")
	testPaths, err := doublestar.FilepathGlob("../../examples/**/devbox.json")
	require.NoError(t, err, "Reading testdata/ should not fail")

	assert.Greater(t, len(testPaths), 0, "testdata/ and examples/ should contain at least 1 test")

	for _, testPath := range testPaths {
		if !strings.Contains(testPath, "/commands/") {
			testShellPlan(t, testPath)
		}
	}
}

func testShellPlan(t *testing.T, testPath string) {
	baseDir := filepath.Dir(testPath)
	testName := fmt.Sprintf("%s_shell_plan", filepath.Base(baseDir))
	t.Run(testName, func(t *testing.T) {
		t.Setenv(envir.XDGDataHome, "/tmp/devbox")
		assert := assert.New(t)

		_, err := Open(&devopt.Opts{
			Dir:    baseDir,
			Stderr: os.Stderr,
		})
		assert.NoErrorf(err, "%s should be a valid devbox project", baseDir)
	})
}

type testNix struct {
	path string
}

func (n *testNix) PrintDevEnv(ctx context.Context, args *nix.PrintDevEnvArgs) (*nix.PrintDevEnvOut, error) {
	return &nix.PrintDevEnvOut{
		Variables: map[string]nix.Variable{
			"PATH": {
				Type:  "exported",
				Value: n.path,
			},
		},
	}, nil
}

func TestComputeEnv(t *testing.T) {
	d := devboxForTesting(t)
	d.nix = &testNix{}
	ctx := t.Context()
	env, err := d.computeEnv(ctx, false /*use cache*/, devopt.EnvOptions{})
	require.NoError(t, err, "computeEnv should not fail")
	assert.NotNil(t, env, "computeEnv should return a valid env")
}

func TestComputeDevboxPathIsIdempotent(t *testing.T) {
	devbox := devboxForTesting(t)
	devbox.nix = &testNix{"/tmp/my/path"}
	ctx := t.Context()
	env, err := devbox.computeEnv(ctx, false /*use cache*/, devopt.EnvOptions{})
	require.NoError(t, err, "computeEnv should not fail")
	path := env["PATH"]
	assert.NotEmpty(t, path, "path should not be nil")

	t.Setenv("PATH", path)
	t.Setenv(envpath.InitPathEnv, env[envpath.InitPathEnv])
	t.Setenv(envpath.PathStackEnv, env[envpath.PathStackEnv])
	t.Setenv(envpath.Key(devbox.ProjectDirHash()), env[envpath.Key(devbox.ProjectDirHash())])

	env, err = devbox.computeEnv(ctx, false /*use cache*/, devopt.EnvOptions{})
	require.NoError(t, err, "computeEnv should not fail")
	path2 := env["PATH"]

	assert.Equal(t, path, path2, "path should be the same")
}

func TestComputeDevboxPathWhenRemoving(t *testing.T) {
	devbox := devboxForTesting(t)
	devbox.nix = &testNix{"/tmp/my/path"}
	ctx := t.Context()
	env, err := devbox.computeEnv(ctx, false /*use cache*/, devopt.EnvOptions{})
	require.NoError(t, err, "computeEnv should not fail")
	path := env["PATH"]
	assert.NotEmpty(t, path, "path should not be nil")
	assert.Contains(t, path, "/tmp/my/path", "path should contain /tmp/my/path")

	t.Setenv("PATH", path)
	t.Setenv(envpath.InitPathEnv, env[envpath.InitPathEnv])
	t.Setenv(envpath.PathStackEnv, env[envpath.PathStackEnv])
	t.Setenv(envpath.Key(devbox.ProjectDirHash()), env[envpath.Key(devbox.ProjectDirHash())])

	devbox.nix.(*testNix).path = ""
	env, err = devbox.computeEnv(ctx, false /*use cache*/, devopt.EnvOptions{})
	require.NoError(t, err, "computeEnv should not fail")
	path2 := env["PATH"]
	assert.NotContains(t, path2, "/tmp/my/path", "path should not contain /tmp/my/path")

	assert.NotEqual(t, path, path2, "path should not be the same")
}

// testNixVars is a nix.Nixer mock whose PrintDevEnv returns a configurable set
// of exported variables. Used to exercise how computeEnv layers the Nix
// dev-env on top of the ambient environment.
type testNixVars struct {
	vars map[string]string
}

func (n *testNixVars) PrintDevEnv(ctx context.Context, args *nix.PrintDevEnvArgs) (*nix.PrintDevEnvOut, error) {
	variables := map[string]nix.Variable{}
	for k, v := range n.vars {
		variables[k] = nix.Variable{Type: "exported", Value: v}
	}
	return &nix.PrintDevEnvOut{Variables: variables}, nil
}

func TestPreserveUserSSLCertFiles(t *testing.T) {
	const userBundle = "/Library/Application Support/Netskope/STAgent/data/nscacert_combined.pem"
	const nixBundle = "/nix/store/abc-nss-cacert-3.108/etc/ssl/certs/ca-bundle.crt"

	t.Run("restores user value when set", func(t *testing.T) {
		env := map[string]string{"NIX_SSL_CERT_FILE": nixBundle, "SSL_CERT_FILE": nixBundle}
		userEnv := map[string]string{"NIX_SSL_CERT_FILE": userBundle, "SSL_CERT_FILE": userBundle}
		preserveUserSSLCertFiles(env, userEnv)
		assert.Equal(t, userBundle, env["NIX_SSL_CERT_FILE"])
		assert.Equal(t, userBundle, env["SSL_CERT_FILE"])
	})

	t.Run("keeps nix value when user did not set one", func(t *testing.T) {
		env := map[string]string{"NIX_SSL_CERT_FILE": nixBundle}
		preserveUserSSLCertFiles(env, map[string]string{})
		assert.Equal(t, nixBundle, env["NIX_SSL_CERT_FILE"])
	})

	t.Run("ignores empty user value", func(t *testing.T) {
		env := map[string]string{"NIX_SSL_CERT_FILE": nixBundle}
		preserveUserSSLCertFiles(env, map[string]string{"NIX_SSL_CERT_FILE": ""})
		assert.Equal(t, nixBundle, env["NIX_SSL_CERT_FILE"])
	})
}

// TestComputeEnvPreservesUserSSLCertFile is a regression test for
// jetify-com/devbox#2604: adding a package that pulls in nss-cacert (e.g.
// httpie) must not clobber a NIX_SSL_CERT_FILE the user set in their own
// environment (e.g. a corporate MITM CA bundle).
func TestComputeEnvPreservesUserSSLCertFile(t *testing.T) {
	const userBundle = "/Library/Application Support/Netskope/STAgent/data/nscacert_combined.pem"
	const nixBundle = "/nix/store/abc-nss-cacert-3.108/etc/ssl/certs/ca-bundle.crt"

	d := devboxForTesting(t)
	d.nix = &testNixVars{vars: map[string]string{
		"PATH":              "/tmp/my/path",
		"NIX_SSL_CERT_FILE": nixBundle,
	}}

	t.Setenv("NIX_SSL_CERT_FILE", userBundle)

	env, err := d.computeEnv(t.Context(), false /*use cache*/, devopt.EnvOptions{})
	require.NoError(t, err, "computeEnv should not fail")
	assert.Equal(t, userBundle, env["NIX_SSL_CERT_FILE"],
		"the user's NIX_SSL_CERT_FILE should win over the nix dev-env value")
}

func devboxForTesting(t *testing.T) *Devbox {
	path := t.TempDir()
	_, err := devconfig.Init(path)
	require.NoError(t, err, "InitConfig should not fail")
	d, err := Open(&devopt.Opts{
		Dir:    path,
		Stderr: os.Stderr,
	})
	require.NoError(t, err, "Open should not fail")

	return d
}
