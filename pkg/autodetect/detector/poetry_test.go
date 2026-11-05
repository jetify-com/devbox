package detector

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoetryDetector_PackagesIgnoresProjectVersion(t *testing.T) {
	dir := t.TempDir()
	// [tool.poetry].version is the project's own version, not Poetry's, so it
	// must not be used to pin the poetry package.
	pyproject := []byte(`
[tool.poetry]
name = "myproject"
version = "1.8.3"
`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pyproject.toml"), pyproject, 0o644))

	d := &PoetryDetector{Root: dir}
	packages, err := d.Packages(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"python@latest", "poetry@latest"}, packages)
}

func TestSanitizeVersion(t *testing.T) {
	tests := map[string]string{
		"3.8":          "3.8",
		"v1.2.3":       "1.2.3",
		"^3.8":         "3.8",
		"~3.8.1":       "3.8.1",
		">=3.8":        "3.8",
		">=3.8,<4.0":   "3.8",
		">=3.8, <4.0":  "3.8",
		"^2.7 || ^3.6": "2.7",
	}
	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			assert.Equal(t, want, sanitizeVersion(input))
		})
	}
}
