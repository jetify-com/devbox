// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirExistsAndIsNotEmpty(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(string) error
		expected bool
	}{
		{
			name: "empty directory",
			setup: func(dir string) error {
				return nil // Directory is already empty
			},
			expected: false,
		},
		{
			name: "directory with files",
			setup: func(dir string) error {
				file := filepath.Join(dir, "test.txt")
				return os.WriteFile(file, []byte("test content"), 0o644)
			},
			expected: true,
		},
		{
			name: "directory with subdirectories",
			setup: func(dir string) error {
				subdir := filepath.Join(dir, "subdir")
				return os.MkdirAll(subdir, 0o755)
			},
			expected: true,
		},
		{
			name: "directory with hidden files",
			setup: func(dir string) error {
				file := filepath.Join(dir, ".hidden")
				return os.WriteFile(file, []byte("hidden content"), 0o644)
			},
			expected: true,
		},
		{
			name: "non-existent directory",
			setup: func(dir string) error {
				return os.RemoveAll(dir)
			},
			expected: false,
		},
	}

	for _, curTest := range tests {
		t.Run(curTest.name, func(t *testing.T) {
			// Create temporary directory for test
			tempDir := t.TempDir()

			// Setup test case
			if curTest.setup != nil {
				err := curTest.setup(tempDir)
				require.NoError(t, err)
			}

			// Run the function
			result := dirExistsAndIsNotEmpty(tempDir)

			// Check results
			assert.Equal(t, curTest.expected, result)
		})
	}
}
