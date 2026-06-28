// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
