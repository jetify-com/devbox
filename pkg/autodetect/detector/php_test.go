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

func TestPHPDetector_Relevance(t *testing.T) {
	tests := []struct {
		name     string
		fs       fstest.MapFS
		expected float64
	}{
		{
			name:     "no composer.json",
			fs:       fstest.MapFS{},
			expected: 0,
		},
		{
			name: "with composer.json",
			fs: fstest.MapFS{
				"composer.json": &fstest.MapFile{
					Data: []byte(`{
						"require": {
							"php": "^8.1"
						}
					}`),
				},
			},
			expected: 1,
		},
	}

	for _, curTest := range tests {
		t.Run(curTest.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, file := range curTest.fs {
				err := os.WriteFile(filepath.Join(dir, name), file.Data, 0o644)
				require.NoError(t, err)
			}

			d := &PHPDetector{Root: dir}
			err := d.Init()
			require.NoError(t, err)

			score, err := d.Relevance(dir)
			require.NoError(t, err)
			assert.Equal(t, curTest.expected, score)
		})
	}
}

func TestPHPDetector_Packages(t *testing.T) {
	tests := []struct {
		name          string
		fs            fstest.MapFS
		expectedPHP   string
		expectedError bool
	}{
		{
			name: "no php version specified",
			fs: fstest.MapFS{
				"composer.json": &fstest.MapFile{
					Data: []byte(`{
						"require": {}
					}`),
				},
			},
			expectedPHP: "php@latest",
		},
		{
			name: "specific php version",
			fs: fstest.MapFS{
				"composer.json": &fstest.MapFile{
					Data: []byte(`{
						"require": {
							"php": "^8.1"
						}
					}`),
				},
			},
			expectedPHP: "php@8.1",
		},
		{
			name: "php version with patch",
			fs: fstest.MapFS{
				"composer.json": &fstest.MapFile{
					Data: []byte(`{
						"require": {
							"php": "^8.1.2"
						}
					}`),
				},
			},
			expectedPHP: "php@8.1.2",
		},
		{
			name: "invalid composer.json",
			fs: fstest.MapFS{
				"composer.json": &fstest.MapFile{
					Data: []byte(`invalid json`),
				},
			},
			expectedError: true,
		},
	}

	for _, curTest := range tests {
		t.Run(curTest.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, file := range curTest.fs {
				err := os.WriteFile(filepath.Join(dir, name), file.Data, 0o644)
				require.NoError(t, err)
			}

			d := &PHPDetector{Root: dir}
			err := d.Init()
			if curTest.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			packages, err := d.Packages(context.Background())
			require.NoError(t, err)
			assert.Equal(t, []string{curTest.expectedPHP}, packages)
		})
	}
}

func TestPHPDetector_PHPExtensions(t *testing.T) {
	tests := []struct {
		name               string
		fs                 fstest.MapFS
		expectedExtensions []string
	}{
		{
			name: "no extensions",
			fs: fstest.MapFS{
				"composer.json": &fstest.MapFile{
					Data: []byte(`{
						"require": {
							"php": "^8.1"
						}
					}`),
				},
			},
			expectedExtensions: []string{},
		},
		{
			name: "multiple extensions",
			fs: fstest.MapFS{
				"composer.json": &fstest.MapFile{
					Data: []byte(`{
						"require": {
							"php": "^8.1",
							"ext-mbstring": "*",
							"ext-imagick": "*"
						}
					}`),
				},
			},
			expectedExtensions: []string{
				"php81Extensions.mbstring@latest",
				"php81Extensions.imagick@latest",
			},
		},
	}

	for _, curTest := range tests {
		t.Run(curTest.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, file := range curTest.fs {
				err := os.WriteFile(filepath.Join(dir, name), file.Data, 0o644)
				require.NoError(t, err)
			}

			d := &PHPDetector{Root: dir}
			err := d.Init()
			require.NoError(t, err)

			extensions, err := d.phpExtensions(context.Background())
			require.NoError(t, err)
			assert.ElementsMatch(t, curTest.expectedExtensions, extensions)
		})
	}
}
