// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.jetify.com/devbox/nix/flake"
)

// TestConfigHashIncludesCreateFilesContent verifies that a plugin's hash
// changes when the content of a create_files source file changes. This is what
// makes local plugins under active development re-create their virtenv files
// when their source changes (https://github.com/jetify-com/devbox/issues/2755).
func TestConfigHashIncludesCreateFilesContent(t *testing.T) {
	pluginDir := t.TempDir()
	projectDir := t.TempDir()

	pluginJSON := `{
		"name": "testplugin",
		"version": "0.0.1",
		"create_files": {
			"{{ .Virtenv }}/test.txt": "test.txt"
		}
	}`
	require.NoError(t, os.WriteFile(
		filepath.Join(pluginDir, "plugin.json"), []byte(pluginJSON), 0o644))
	srcFile := filepath.Join(pluginDir, "test.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("123"), 0o644))

	cfg := localPluginConfigForTest(t, pluginDir, projectDir)

	hash1, err := cfg.Hash()
	require.NoError(t, err)

	// Re-hashing without any change must be stable.
	hash1Again, err := cfg.Hash()
	require.NoError(t, err)
	assert.Equal(t, hash1, hash1Again, "hash should be stable when nothing changes")

	// Changing the create_files source content must change the hash so that the
	// file gets re-created in the virtenv on the next shell.
	require.NoError(t, os.WriteFile(srcFile, []byte("456"), 0o644))
	hash2, err := cfg.Hash()
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hash2,
		"hash should change when create_files source content changes")
}

func localPluginConfigForTest(t *testing.T, pluginDir, projectDir string) *Config {
	t.Helper()
	ref, err := flake.ParseRef("path:" + pluginDir)
	require.NoError(t, err)
	localPlugin, err := newLocalPlugin(ref, projectDir)
	require.NoError(t, err)
	cfg, err := getConfigIfAny(localPlugin, projectDir)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	return cfg
}
