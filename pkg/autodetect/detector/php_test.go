package detector

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPHPDetector_Relevance(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		expected float64
	}{
		{
			name: "no composer.json",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				return dir
			},
			expected: 1,
		},
		{
			name: "with composer.json",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(`{
					"require": {
						"php": "^8.1"
					}
				}`), 0644)
				require.NoError(t, err)
				return dir
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			d := &PHPDetector{Root: dir}
			err := d.Init()
			require.NoError(t, err)

			score, err := d.Relevance(dir)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, score)
		})
	}
}

func TestPHPDetector_Packages(t *testing.T) {
	tests := []struct {
		name          string
		composerJSON  string
		expectedPHP   string
		expectedError bool
	}{
		{
			name: "no php version specified",
			composerJSON: `{
				"require": {}
			}`,
			expectedPHP: "php@latest",
		},
		{
			name: "specific php version",
			composerJSON: `{
				"require": {
					"php": "^8.1"
				}
			}`,
			expectedPHP: "php@8.1",
		},
		{
			name: "php version with patch",
			composerJSON: `{
				"require": {
					"php": "^8.1.2"
				}
			}`,
			expectedPHP: "php@8.1.2",
		},
		{
			name:          "invalid composer.json",
			composerJSON:  `invalid json`,
			expectedError: true,
		},
	}

	for _, curTest := range tests {
		t.Run(curTest.name, func(t *testing.T) {
			dir := t.TempDir()
			if curTest.composerJSON != "" {
				err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(curTest.composerJSON), 0644)
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
		composerJSON       string
		expectedExtensions []string
	}{
		{
			name: "no extensions",
			composerJSON: `{
				"require": {
					"php": "^8.1"
				}
			}`,
			expectedExtensions: []string{},
		},
		{
			name: "multiple extensions",
			composerJSON: `{
				"require": {
					"ext-mbstring": "*",
					"ext-imagick": "*"
				}
			}`,
			expectedExtensions: []string{
				"phpExtensions.mbstring",
				"phpExtensions.imagick",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(tt.composerJSON), 0644)
			require.NoError(t, err)

			d := &PHPDetector{Root: dir}
			err = d.Init()
			require.NoError(t, err)

			extensions, err := d.phpExtensions()
			require.NoError(t, err)
			assert.ElementsMatch(t, tt.expectedExtensions, extensions)
		})
	}
}
