// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"
)

const processComposeModulePath = "github.com/f1bonacc1/process-compose"

// TestProcessComposeVersionMatchesGoMod guards against drift between the
// process-compose binary we install into the global utility project and the
// process-compose library we compile against. The two are separate pins that
// nothing else keeps in sync, and they silently diverged once already: go.mod
// sat at v1.64.1 while processComposeVersion had moved on to 1.110.0.
//
// Skew matters because internal/services decodes the running daemon's REST
// responses into the library's types (types.ProcessesState and friends). If the
// two versions disagree about a JSON tag, `devbox services ls` misreports state
// instead of failing loudly.
func TestProcessComposeVersionMatchesGoMod(t *testing.T) {
	goModPath := findGoMod(t)

	data, err := os.ReadFile(goModPath)
	require.NoError(t, err, "read %s", goModPath)

	f, err := modfile.Parse(goModPath, data, nil)
	require.NoError(t, err, "parse %s", goModPath)

	var got string
	for _, r := range f.Require {
		if r.Mod.Path == processComposeModulePath {
			got = r.Mod.Version
			break
		}
	}
	require.NotEmpty(t, got, "%s not found in %s", processComposeModulePath, goModPath)

	want := "v" + processComposeVersion
	require.Equal(t, want, got,
		"process-compose version skew: processComposeVersion is %q (internal/devbox/util.go) "+
			"but go.mod requires %s %s. Bump both to the same version.",
		processComposeVersion, processComposeModulePath, got)
}

// findGoMod walks up from the working directory to locate the module's go.mod,
// so the test keeps working if this package is ever moved.
func findGoMod(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		candidate := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod found above %s", dir)
		}
		dir = parent
	}
}
