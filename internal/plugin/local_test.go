package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"go.jetify.com/devbox/nix/flake"
)

// newTestLocalPlugin writes a plugin.json (with the given create_files content
// file) into a temp dir and returns a *LocalPlugin pointing at it along with the
// path of the referenced content file.
func newTestLocalPlugin(t *testing.T, fileContents string) (*LocalPlugin, string) {
	t.Helper()
	pluginDir := t.TempDir()

	pluginJSON := `{
		"name": "testplugin",
		"create_files": {
			"{{ .Virtenv }}/test.txt": "test.txt"
		}
	}`
	if err := os.WriteFile(filepath.Join(pluginDir, pluginConfigName), []byte(pluginJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	contentPath := filepath.Join(pluginDir, "test.txt")
	if err := os.WriteFile(contentPath, []byte(fileContents), 0o644); err != nil {
		t.Fatal(err)
	}

	local, err := newLocalPlugin(flake.Ref{Type: flake.TypePath, Path: pluginDir}, pluginDir)
	if err != nil {
		t.Fatal(err)
	}
	return local, contentPath
}

// TestLocalPluginHashIncludesCreateFilesContent verifies that editing a file
// referenced by a local plugin's create_files changes the plugin config hash,
// even though devbox.json and the plugin.json are unchanged. This is what allows
// Devbox to detect the change and regenerate the file in the virtenv.
// See https://github.com/jetify-com/devbox/issues/2755.
func TestLocalPluginHashIncludesCreateFilesContent(t *testing.T) {
	local, contentPath := newTestLocalPlugin(t, "123")

	pluginJSON, err := local.Fetch()
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := buildConfig(local, filepath.Dir(contentPath), string(pluginJSON))
	if err != nil {
		t.Fatal(err)
	}

	before, err := cfg.Hash()
	if err != nil {
		t.Fatal(err)
	}

	// Edit the referenced content file without touching plugin.json.
	if err := os.WriteFile(contentPath, []byte("456"), 0o644); err != nil {
		t.Fatal(err)
	}

	after, err := cfg.Hash()
	if err != nil {
		t.Fatal(err)
	}

	if before == after {
		t.Errorf("hash did not change after editing create_files content: %q", before)
	}
}

// TestLocalPluginHashStableWithoutContentChange verifies the hash is stable when
// the referenced files don't change.
func TestLocalPluginHashStableWithoutContentChange(t *testing.T) {
	local, contentPath := newTestLocalPlugin(t, "123")

	pluginJSON, err := local.Fetch()
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := buildConfig(local, filepath.Dir(contentPath), string(pluginJSON))
	if err != nil {
		t.Fatal(err)
	}

	first, err := cfg.Hash()
	if err != nil {
		t.Fatal(err)
	}
	second, err := cfg.Hash()
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Errorf("hash is not stable: %q != %q", first, second)
	}
}
