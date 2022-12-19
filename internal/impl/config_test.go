package impl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindConfigDirFromParentDirSearch(t *testing.T) {
	testCases := []struct {
		name        string
		allDirs     string
		configDir   string
		searchPath  string
		expectError bool
	}{
		{
			name:        "search_dir_same_as_config_dir",
			allDirs:     "a/b/c",
			configDir:   "a/b",
			searchPath:  "a/b",
			expectError: false,
		},
		{
			name:        "search_dir_in_nested_folder",
			allDirs:     "a/b/c",
			configDir:   "a/b",
			searchPath:  "a/b/c",
			expectError: false,
		},
		{
			name:        "search_dir_in_parent_folder",
			allDirs:     "a/b/c",
			configDir:   "a/b",
			searchPath:  "a",
			expectError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert := assert.New(t)

			root, err := filepath.Abs(t.TempDir())
			assert.NoError(err)

			err = os.MkdirAll(filepath.Join(root, testCase.allDirs), 0777)
			assert.NoError(err)

			absConfigPath, err := filepath.Abs(filepath.Join(root, testCase.configDir, configFilename))
			assert.NoError(err)
			err = os.WriteFile(absConfigPath, []byte("{}"), 0666)
			assert.NoError(err)

			absSearchPath := filepath.Join(root, testCase.searchPath)
			result, err := findConfigDirFromParentDirSearch(root, absSearchPath)

			if testCase.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(filepath.Dir(filepath.Join(absConfigPath)), result)
			}
		})
	}
}

func TestFindConfigDirAtPath(t *testing.T) {

	testCases := []struct {
		name        string
		allDirs     string
		configDir   string
		flagPath    string
		expectError bool
	}{
		{
			name:        "flag_path_is_dir_has_config",
			allDirs:     "a/b/c",
			configDir:   "a/b",
			flagPath:    "a/b",
			expectError: false,
		},
		{
			name:        "flag_path_is_dir_missing_config",
			allDirs:     "a/b/c",
			configDir:   "", // missing config
			flagPath:    "a/b",
			expectError: true,
		},
		{
			name:        "flag_path_is_file_has_config",
			allDirs:     "a/b/c",
			configDir:   "a/b",
			flagPath:    "a/b/" + configFilename,
			expectError: false,
		},
		{
			name:        "flag_path_is_file_missing_config",
			allDirs:     "a/b/c",
			configDir:   "", // missing config
			flagPath:    "a/b/" + configFilename,
			expectError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert := assert.New(t)

			root, err := filepath.Abs(t.TempDir())
			assert.NoError(err)

			err = os.MkdirAll(filepath.Join(root, testCase.allDirs), 0777)
			assert.NoError(err)

			var absConfigPath string
			if testCase.configDir != "" {
				absConfigPath, err = filepath.Abs(filepath.Join(root, testCase.configDir, configFilename))
				assert.NoError(err)
				err = os.WriteFile(absConfigPath, []byte("{}"), 0666)
				assert.NoError(err)
			}

			absFlagPath := filepath.Join(root, testCase.flagPath)
			result, err := findConfigDirAtPath(absFlagPath)

			if testCase.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(filepath.Dir(filepath.Join(absConfigPath)), result)
			}
		})
	}
}

func TestNixpkgsValidation(t *testing.T) {
	testCases := map[string]struct {
		commit   string
		isErrant bool
	}{
		"invalid_nixpkg_commit": {"1234545", true},
		"valid_nixpkg_commit":   {"af9e00071d0971eb292fd5abef334e66eda3cb69", false},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			err := validateNixpkg(&Config{
				Nixpkgs: NixpkgsConfig{
					Commit: testCase.commit,
				},
			})
			if testCase.isErrant {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}
