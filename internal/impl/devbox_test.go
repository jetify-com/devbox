// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

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
	"go.jetpack.io/devbox/internal/impl/envpath"

	"go.jetpack.io/devbox/internal/devconfig"
	"go.jetpack.io/devbox/internal/envir"
	"go.jetpack.io/devbox/internal/impl/devopt"
	"go.jetpack.io/devbox/internal/nix"
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
			Pure:   false,
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

func TestComputeNixEnv(t *testing.T) {
	d := devboxForTesting(t)
	d.nix = &testNix{}
	ctx := context.Background()
	env, err := d.computeNixEnv(ctx, false /*use cache*/)
	require.NoError(t, err, "computeNixEnv should not fail")
	assert.NotNil(t, env, "computeNixEnv should return a valid env")
}

func TestComputeNixPathIsIdempotent(t *testing.T) {
	devbox := devboxForTesting(t)
	devbox.nix = &testNix{"/tmp/my/path"}
	ctx := context.Background()
	env, err := devbox.computeNixEnv(ctx, false /*use cache*/)
	require.NoError(t, err, "computeNixEnv should not fail")
	path := env["PATH"]
	assert.NotEmpty(t, path, "path should not be nil")

	t.Setenv("PATH", path)
	t.Setenv(envpath.InitPathEnv, env[envpath.InitPathEnv])
	t.Setenv(envpath.PathStackEnv, env[envpath.PathStackEnv])
	t.Setenv(envpath.Key(devbox.projectDirHash()), env[envpath.Key(devbox.projectDirHash())])

	env, err = devbox.computeNixEnv(ctx, false /*use cache*/)
	require.NoError(t, err, "computeNixEnv should not fail")
	path2 := env["PATH"]

	assert.Equal(t, path, path2, "path should be the same")
}

func TestComputeNixPathWhenRemoving(t *testing.T) {
	devbox := devboxForTesting(t)
	devbox.nix = &testNix{"/tmp/my/path"}
	ctx := context.Background()
	env, err := devbox.computeNixEnv(ctx, false /*use cache*/)
	require.NoError(t, err, "computeNixEnv should not fail")
	path := env["PATH"]
	assert.NotEmpty(t, path, "path should not be nil")
	assert.Contains(t, path, "/tmp/my/path", "path should contain /tmp/my/path")

	t.Setenv("PATH", path)
	t.Setenv(envpath.InitPathEnv, env[envpath.InitPathEnv])
	t.Setenv(envpath.PathStackEnv, env[envpath.PathStackEnv])
	t.Setenv(envpath.Key(devbox.projectDirHash()), env[envpath.Key(devbox.projectDirHash())])

	devbox.nix.(*testNix).path = ""
	env, err = devbox.computeNixEnv(ctx, false /*use cache*/)
	require.NoError(t, err, "computeNixEnv should not fail")
	path2 := env["PATH"]
	assert.NotContains(t, path2, "/tmp/my/path", "path should not contain /tmp/my/path")

	assert.NotEqual(t, path, path2, "path should not be the same")
}

func devboxForTesting(t *testing.T) *Devbox {
	path := t.TempDir()
	_, err := devconfig.Init(path, os.Stdout)
	require.NoError(t, err, "InitConfig should not fail")
	d, err := Open(&devopt.Opts{
		Dir:    path,
		Stderr: os.Stderr,
		Pure:   false,
	})
	require.NoError(t, err, "Open should not fail")

	return d
}
