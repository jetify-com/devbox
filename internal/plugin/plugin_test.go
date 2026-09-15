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

// fakeIncludable is a minimal Includable used to exercise buildConfig's
// template-placeholder substitution without touching the filesystem or network.
type fakeIncludable struct {
	name string
}

func (f fakeIncludable) CanonicalName() string              { return f.name }
func (f fakeIncludable) FileContent(string) ([]byte, error) { return nil, nil }
func (f fakeIncludable) Hash() string                       { return "" }
func (f fakeIncludable) LockfileKey() string                { return f.name }

// TestBuildConfigTemplatePlaceholders documents and locks in the meaning of the
// template placeholders available to plugins. In particular it guards against
// the confusion reported in #1987: DevboxDirRoot is the root of the devbox.d
// directory (not the project root), and DevboxProjectDir is the project root.
func TestBuildConfigTemplatePlaceholders(t *testing.T) {
	projectDir := filepath.Join("/home", "user", "my-project")
	const pluginName = "my-plugin"

	content := `{
  "name": "my-plugin",
  "version": "0.0.1",
  "create_files": {
    "{{ .DevboxProjectDir }}/project-root": "a",
    "{{ .DevboxDirRoot }}/dir-root": "b",
    "{{ .DevboxDir }}/dir": "c",
    "{{ .Virtenv }}/virtenv": "d"
  }
}`

	cfg, err := buildConfig(fakeIncludable{name: pluginName}, projectDir, content)
	require.NoError(t, err)

	// Invert the create_files map (contentPath -> renderedPath) so the
	// assertions read naturally regardless of map ordering.
	renderedByContent := map[string]string{}
	for renderedPath, contentPath := range cfg.CreateFiles {
		renderedByContent[contentPath] = renderedPath
	}

	assert.Equal(t,
		filepath.Join(projectDir, "project-root"),
		renderedByContent["a"],
		"DevboxProjectDir should be the project root (where devbox.json lives)",
	)
	assert.Equal(t,
		filepath.Join(projectDir, devboxDirName, "dir-root"),
		renderedByContent["b"],
		"DevboxDirRoot should be <projectDir>/devbox.d",
	)
	assert.Equal(t,
		filepath.Join(projectDir, devboxDirName, pluginName, "dir"),
		renderedByContent["c"],
		"DevboxDir should be <projectDir>/devbox.d/<plugin.name>",
	)
	assert.Equal(t,
		filepath.Join(projectDir, VirtenvPath, pluginName, "virtenv"),
		renderedByContent["d"],
		"Virtenv should be <projectDir>/.devbox/virtenv/<plugin.name>",
	)
}
