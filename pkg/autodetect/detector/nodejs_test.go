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
		expectedEnv      map[string]string
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
			expectedEnv:      map[string]string{"DEVBOX_COREPACK_ENABLED": "1"},
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
			expectedEnv:      map[string]string{"DEVBOX_COREPACK_ENABLED": "1"},
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
			expectedEnv:      map[string]string{},
		},
		{
			name:             "empty directory",
			fs:               fstest.MapFS{},
			expected:         0,
			expectedPackages: []string{},
			expectedEnv:      map[string]string{},
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

			detector := &NodeJSDetector{Root: dir}
			err := detector.Init()
			require.NoError(t, err)

			score, err := detector.Relevance(dir)
			require.NoError(t, err)
			assert.Equal(t, curTest.expected, score)
			if score > 0 {
				packages, err := detector.Packages(context.Background())
				require.NoError(t, err)
				assert.Equal(t, curTest.expectedPackages, packages)

				env, err := detector.Env(context.Background())
				require.NoError(t, err)
				assert.Equal(t, curTest.expectedEnv, env)
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

func TestNodeJSDetector_Env(t *testing.T) {
	d := &NodeJSDetector{}
	env, err := d.Env(context.Background())
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"DEVBOX_COREPACK_ENABLED": "1"}, env)
}
