// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.jetpack.io/devbox/internal/devconfig/configfile"
)

func TestFindProjectDirFromParentDirSearch(t *testing.T) {
	testCases := []struct {
		name        string
		allDirs     string
		projectDir  string
		searchPath  string
		expectError bool
	}{
		{
			name:        "search_dir_same_as_config_dir",
			allDirs:     "a/b/c",
			projectDir:  "a/b",
			searchPath:  "a/b",
			expectError: false,
		},
		{
			name:        "search_dir_in_nested_folder",
			allDirs:     "a/b/c",
			projectDir:  "a/b",
			searchPath:  "a/b/c",
			expectError: false,
		},
		{
			name:        "search_dir_in_parent_folder",
			allDirs:     "a/b/c",
			projectDir:  "a/b",
			searchPath:  "a",
			expectError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert := assert.New(t)

			root, err := filepath.Abs(t.TempDir())
			assert.NoError(err)

			err = os.MkdirAll(filepath.Join(root, testCase.allDirs), 0o777)
			assert.NoError(err)

			absProjectPath, err := filepath.Abs(filepath.Join(root, testCase.projectDir, configfile.DefaultName))
			assert.NoError(err)
			err = os.WriteFile(absProjectPath, []byte("{}"), 0o666)
			assert.NoError(err)

			absSearchPath := filepath.Join(root, testCase.searchPath)
			result, err := findProjectDirFromParentDirSearch(root, absSearchPath)

			if testCase.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(filepath.Dir(filepath.Join(absProjectPath)), result)
			}
		})
	}
}

func TestFindParentDirAtPath(t *testing.T) {
	testCases := []struct {
		name        string
		allDirs     string
		projectDir  string
		flagPath    string
		expectError bool
	}{
		{
			name:        "flag_path_is_dir_has_config",
			allDirs:     "a/b/c",
			projectDir:  "a/b",
			flagPath:    "a/b",
			expectError: false,
		},
		{
			name:        "flag_path_is_dir_missing_config",
			allDirs:     "a/b/c",
			projectDir:  "", // missing config
			flagPath:    "a/b",
			expectError: true,
		},
		{
			name:        "flag_path_is_file_has_config",
			allDirs:     "a/b/c",
			projectDir:  "a/b",
			flagPath:    "a/b/" + configfile.DefaultName,
			expectError: false,
		},
		{
			name:        "flag_path_is_file_missing_config",
			allDirs:     "a/b/c",
			projectDir:  "", // missing config
			flagPath:    "a/b/" + configfile.DefaultName,
			expectError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert := assert.New(t)

			root, err := filepath.Abs(t.TempDir())
			assert.NoError(err)

			err = os.MkdirAll(filepath.Join(root, testCase.allDirs), 0o777)
			assert.NoError(err)

			var absProjectPath string
			if testCase.projectDir != "" {
				absProjectPath, err = filepath.Abs(filepath.Join(root, testCase.projectDir, configfile.DefaultName))
				assert.NoError(err)
				err = os.WriteFile(absProjectPath, []byte("{}"), 0o666)
				assert.NoError(err)
			}

			absFlagPath := filepath.Join(root, testCase.flagPath)
			result, err := findProjectDirAtPath(absFlagPath)

			if testCase.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(filepath.Dir(filepath.Join(absProjectPath)), result)
			}
		})
	}
}
