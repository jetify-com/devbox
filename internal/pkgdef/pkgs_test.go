// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// networkTestsEnv enables tests that resolve the shipped package
// definitions against the real vendors.
const networkTestsEnv = "DEVBOX_RUN_PKGDEF_TESTS"

// shippedDefinitions returns the package definitions in pkgs/, keyed by
// package directory name.
func shippedDefinitions(t *testing.T) map[string]string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("..", "..", "pkgs", "*", "*.pkg.json*"))
	require.NoError(t, err)
	require.NotEmpty(t, paths)
	defs := map[string]string{}
	for _, p := range paths {
		defs[filepath.Base(filepath.Dir(p))] = p
	}
	return defs
}

func loadShipped(t *testing.T, path string) *Definition {
	t.Helper()
	loaded, err := loadLocal(Ref{Source: path}, "")
	require.NoError(t, err)
	return loaded.Definition
}

// TestShippedDefinitionsParse checks that every definition in pkgs/ is
// valid, is named after its directory and file, and supports every
// platform Devbox does.
func TestShippedDefinitionsParse(t *testing.T) {
	for dir, path := range shippedDefinitions(t) {
		t.Run(dir, func(t *testing.T) {
			def := loadShipped(t, path)
			assert.Equal(t, dir, def.Name, "definition name matches its directory")
			assert.Contains(t, []string{dir + ".pkg.json", dir + ".pkg.jsonc"}, filepath.Base(path))
			assert.ElementsMatch(t, knownSystems, def.Platforms)
			assert.NotEmpty(t, def.Binaries, "binaries lets install verify the result")
			assert.NotEmpty(t, def.Homepage)
			assert.NotEmpty(t, def.License)
		})
	}
}

// TestShippedDefinitionsResolve resolves the latest version of every
// definition in pkgs/ against the real vendor and locks it for every
// platform. It catches vendors that changed their URLs or metadata.
func TestShippedDefinitionsResolve(t *testing.T) {
	if on, _ := strconv.ParseBool(os.Getenv(networkTestsEnv)); !on {
		t.Skipf("Skipping network test. To enable, set %s=1.", networkTestsEnv)
	}
	client := NewClient()
	client.CacheDir = t.TempDir()
	for dir, path := range shippedDefinitions(t) {
		t.Run(dir, func(t *testing.T) {
			def := loadShipped(t, path)
			resolved, err := client.Resolve(t.Context(), def, Latest)
			require.NoError(t, err)
			for system, artifact := range resolved.Artifacts {
				assert.NotEmpty(t, artifact.URL, system)
				assert.True(t, artifact.HasDigest(), "%s: the latest version should have a vendor checksum", system)
			}
			// Vendors drop platforms (Temurin 27 has no Intel macOS build),
			// so only the platform running the test is required.
			assert.Contains(t, resolved.Artifacts, hostSystem(t))
			var missing []string
			for _, system := range knownSystems {
				if _, ok := resolved.Artifacts[system]; !ok {
					missing = append(missing, system)
				}
			}
			t.Logf("%s@latest = %s (not published for: %v)", def.Name, resolved.Version, missing)
		})
	}
}

// hostSystem returns the nix system of the machine running the test.
func hostSystem(t *testing.T) string {
	t.Helper()
	arch := map[string]string{"amd64": "x86_64", "arm64": "aarch64"}[runtime.GOARCH]
	require.NotEmpty(t, arch, "unsupported GOARCH %s", runtime.GOARCH)
	return arch + "-" + runtime.GOOS
}
