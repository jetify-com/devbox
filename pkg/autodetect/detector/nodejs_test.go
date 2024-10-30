package detector

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeJSDetector_Relevance(t *testing.T) {
	tests := []struct {
		name             string
		fs               fstest.MapFS
		expected         float64
		expectedPackages []string
	}{
		{
			name: "package.json in root",
			fs: fstest.MapFS{
				"package.json": &fstest.MapFile{
					Data: []byte(`{}`),
				},
			},
			expected:         1,
			expectedPackages: []string{"nodejs@latest"},
		},
		{
			name: "package.json with node version",
			fs: fstest.MapFS{
				"package.json": &fstest.MapFile{
					Data: []byte(`{
						"engines": {
							"node": ">=18.0.0"
						}
					}`),
				},
			},
			expected:         1,
			expectedPackages: []string{"nodejs@18.0.0"},
		},
		{
			name: "no nodejs files",
			fs: fstest.MapFS{
				"main.py": &fstest.MapFile{
					Data: []byte(``),
				},
				"requirements.txt": &fstest.MapFile{
					Data: []byte(``),
				},
			},
			expected:         0,
			expectedPackages: []string{},
		},
		{
			name:             "empty directory",
			fs:               fstest.MapFS{},
			expected:         0,
			expectedPackages: []string{},
		},
	}

	for _, curTest := range tests {
		t.Run(curTest.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, file := range curTest.fs {
				fullPath := filepath.Join(dir, name)
				err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
				require.NoError(t, err)
				err = os.WriteFile(fullPath, file.Data, 0o644)
				require.NoError(t, err)
			}

			d := &NodeJSDetector{Root: dir}
			err := d.Init()
			require.NoError(t, err)

			score, err := d.Relevance(dir)
			require.NoError(t, err)
			assert.Equal(t, curTest.expected, score)
			if score > 0 {
				packages, err := d.Packages(context.Background())
				require.NoError(t, err)
				assert.Equal(t, curTest.expectedPackages, packages)
			}
		})
	}
}

func TestNodeJSDetector_Packages(t *testing.T) {
	d := &NodeJSDetector{}
	packages, err := d.Packages(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"nodejs@latest"}, packages)
}
