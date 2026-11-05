// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.jetify.com/devbox/internal/devbox/devopt"
	"go.jetify.com/devbox/internal/lock"
	"go.jetify.com/devbox/internal/nix"
	"go.jetify.com/devbox/internal/shellgen"
)

// cacheOnlyNix serves print-dev-env from the cache and fails if devbox tries
// to run `nix print-dev-env`.
type cacheOnlyNix struct {
	uncachedCalls int
}

func (n *cacheOnlyNix) PrintDevEnv(_ context.Context, args *nix.PrintDevEnvArgs) (*nix.PrintDevEnvOut, error) {
	if !args.UsePrintDevEnvCache {
		n.uncachedCalls++
		return nil, errors.New("unexpected uncached nix print-dev-env call")
	}
	return &nix.PrintDevEnvOut{Variables: map[string]nix.Variable{
		"PATH": {Type: "exported", Value: "/nix/store/fake/bin"},
	}}, nil
}

const stateTestConfig = `{
  "packages": ["github:NixOS/nixpkgs/nixpkgs-unstable#hello"],
  "env": {"MY_VAR": "old"},
  "shell": {
    "init_hook": ["echo old-hook"],
    "scripts": {"old-script": "echo old"}
  }
}`

func openStateTestProject(t *testing.T, dir, config string) *Devbox {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "devbox.json"), []byte(config), 0o644))
	d, err := Open(&devopt.Opts{Dir: dir, Stderr: os.Stderr})
	require.NoError(t, err)
	return d
}

func stateStatus(t *testing.T, d *Devbox) lock.StateStatus {
	t.Helper()
	status, err := d.lockfile.StateStatus(isFishShell())
	require.NoError(t, err)
	return status
}

func TestEnsureStateNonNixChangeSkipsNix(t *testing.T) {
	dir := t.TempDir()
	box := openStateTestProject(t, dir, stateTestConfig)

	// Simulate a previous full recompute.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".devbox"), 0o755))
	require.NoError(t, box.saveStateHashFile())
	require.Equal(t, lock.StateUpToDate, stateStatus(t, box))

	// Edit env, the init hook and the scripts.
	box = openStateTestProject(t, dir, `{
  "packages": ["github:NixOS/nixpkgs/nixpkgs-unstable#hello"],
  "env": {"MY_VAR": "new"},
  "shell": {
    "init_hook": ["echo new-hook"],
    "scripts": {"new-script": "echo new"}
  }
}`)
	fakeNix := &cacheOnlyNix{}
	box.nix = fakeNix
	require.Equal(t, lock.StateNonNixStale, stateStatus(t, box))

	ctx := t.Context()
	require.NoError(t, box.ensureStateIsUpToDate(ctx, ensure))
	assert.Equal(t, 0, fakeNix.uncachedCalls, "ensureStateIsUpToDate should not run nix")
	assert.Equal(t, lock.StateUpToDate, stateStatus(t, box))

	hooks, err := os.ReadFile(shellgen.ScriptPath(dir, shellgen.HooksFilename))
	require.NoError(t, err)
	assert.Contains(t, string(hooks), "new-hook")
	assert.FileExists(t, shellgen.ScriptPath(dir, "new-script"))
	assert.NoFileExists(t, shellgen.ScriptPath(dir, "old-script"))

	env, err := box.computeEnv(ctx, true /*usePrintDevEnvCache*/, devopt.EnvOptions{})
	require.NoError(t, err)
	assert.Equal(t, "new", env["MY_VAR"])
	assert.Equal(t, 0, fakeNix.uncachedCalls)
}

func TestSkipRecomputeRefreshesNonNixState(t *testing.T) {
	dir := t.TempDir()
	box := openStateTestProject(t, dir, stateTestConfig)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".devbox"), 0o755))
	require.NoError(t, box.saveStateHashFile())

	box = openStateTestProject(t, dir, `{
  "packages": ["github:NixOS/nixpkgs/nixpkgs-unstable#hello"],
  "env": {"MY_VAR": "new"},
  "shell": {"init_hook": ["echo new-hook"]}
}`)
	fakeNix := &cacheOnlyNix{}
	box.nix = fakeNix

	staleCalls := 0
	env, err := box.ensureStateIsUpToDateAndComputeEnv(t.Context(), devopt.EnvOptions{
		SkipRecompute: true,
		Hooks:         devopt.LifecycleHooks{OnStaleState: func() { staleCalls++ }},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, staleCalls, "non-nix changes should be refreshed, not reported as stale")
	assert.Equal(t, 0, fakeNix.uncachedCalls)
	assert.Equal(t, "new", env["MY_VAR"])
	assert.Equal(t, lock.StateUpToDate, stateStatus(t, box))

	hooks, err := os.ReadFile(shellgen.ScriptPath(dir, shellgen.HooksFilename))
	require.NoError(t, err)
	assert.Contains(t, string(hooks), "new-hook")
}

func TestStateStatusNixChanges(t *testing.T) {
	tests := []struct {
		name   string
		config string
	}{
		{"packages", `{
  "packages": ["github:NixOS/nixpkgs/nixpkgs-unstable#cowsay"],
  "env": {"MY_VAR": "old"},
  "shell": {"init_hook": ["echo old-hook"], "scripts": {"old-script": "echo old"}}
}`},
		{"packages and env", `{
  "packages": ["github:NixOS/nixpkgs/nixpkgs-unstable#hello", "github:NixOS/nixpkgs/nixpkgs-unstable#cowsay"],
  "env": {"MY_VAR": "new"},
  "shell": {"init_hook": ["echo old-hook"], "scripts": {"old-script": "echo old"}}
}`},
		{"include", `{
  "packages": ["github:NixOS/nixpkgs/nixpkgs-unstable#hello"],
  "env": {"MY_VAR": "old"},
  "shell": {"init_hook": ["echo old-hook"], "scripts": {"old-script": "echo old"}},
  "include": ["plugin:nginx"]
}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			box := openStateTestProject(t, dir, stateTestConfig)
			require.NoError(t, os.MkdirAll(filepath.Join(dir, ".devbox"), 0o755))
			require.NoError(t, box.saveStateHashFile())

			box = openStateTestProject(t, dir, tt.config)
			assert.Equal(t, lock.StateStale, stateStatus(t, box))
		})
	}
}
