package devbox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func TestConfigShellCmdsUnmarshalString(t *testing.T) {
	tests := []struct {
		jsonIn string
		want   ConfigShellCmds
	}{
		{
			jsonIn: `null`,
			want: ConfigShellCmds{
				MarshalAs: CmdArray,
				Cmds:      nil,
			},
		},
		{
			jsonIn: `""`,
			want: ConfigShellCmds{
				MarshalAs: CmdString,
				Cmds:      []string{""},
			},
		},
		{
			jsonIn: `"\n"`,
			want: ConfigShellCmds{
				MarshalAs: CmdString,
				Cmds:      []string{"\n"},
			},
		},
		{
			jsonIn: `"echo 'line1'\necho 'line2'"`,
			want: ConfigShellCmds{
				MarshalAs: CmdString,
				Cmds:      []string{"echo 'line1'\necho 'line2'"},
			},
		},
		{
			jsonIn: `"echo '\nline1'\necho 'line2'\n"`,
			want: ConfigShellCmds{
				MarshalAs: CmdString,
				Cmds:      []string{"echo '\nline1'\necho 'line2'\n"},
			},
		},
		{
			jsonIn: `"echo 'line1'\n\necho 'line2'"`,
			want: ConfigShellCmds{
				MarshalAs: CmdString,
				Cmds:      []string{"echo 'line1'\n\necho 'line2'"},
			},
		},
		{
			jsonIn: `"echo 'line1'\necho '\tline2'"`,
			want: ConfigShellCmds{
				MarshalAs: CmdString,
				Cmds:      []string{"echo 'line1'\necho '\tline2'"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.jsonIn, func(t *testing.T) {
			got := ConfigShellCmds{}
			if err := json.Unmarshal([]byte(test.jsonIn), &got); err != nil {
				t.Fatal("Got error unmarshalling test input:", err)
			}
			if got.MarshalAs != test.want.MarshalAs {
				t.Errorf("Got MarshalAs == %s after unmarshalling, want "+
					"MarshalAs == %s.", got.MarshalAs, test.want.MarshalAs)
			}
			if len(got.Cmds) != len(test.want.Cmds) {
				t.Fatalf("len(got.Cmds) != len(want.Cmds)\ngot:  %q (len %d)\nwant: %q (len %d)",
					got.Cmds, len(got.Cmds), test.want.Cmds, len(test.want.Cmds))
			}
			for i := range got.Cmds {
				got, want := got.Cmds[i], test.want.Cmds[i]
				if got != want {
					t.Fatalf("got.Cmds[%[1]d] != want.Cmds[%[1]d]\ngot:  %q\nwant: %q",
						i, got, want)
				}
			}
			b, err := json.Marshal(got)
			if err != nil {
				t.Fatal("Got error marshalling back to JSON:", err)
			}
			if diff := cmp.Diff(test.jsonIn, string(b)); diff != "" {
				t.Errorf("Got different JSON after unmarshalling and re-marshalling (-want +got):\n%s", diff)
			}
		})
	}
}

func TestConfigShellCmdsString(t *testing.T) {
	tests := []struct {
		jsonIn string
		want   string
	}{
		{
			jsonIn: `null`,
			want:   "",
		},
		{
			jsonIn: `[]`,
			want:   "",
		},
		{
			jsonIn: `[""]`,
			want:   "",
		},
		{
			jsonIn: `["\n"]`,
			want:   "\n",
		},
		{
			jsonIn: `["echo 'line1'\necho 'line2'"]`,
			want:   "echo 'line1'\necho 'line2'",
		},
		{
			jsonIn: `["echo 'line1'", "echo 'line2'"]`,
			want:   "echo 'line1'\necho 'line2'",
		},
		{
			jsonIn: `["echo 'line1'\n\necho 'line2'"]`,
			want:   "echo 'line1'\n\necho 'line2'",
		},
		{
			jsonIn: `["echo 'line1'", "", "echo 'line2'"]`,
			want:   "echo 'line1'\n\necho 'line2'",
		},
	}
	for _, test := range tests {
		t.Run(test.jsonIn, func(t *testing.T) {
			got := ConfigShellCmds{}
			if err := json.Unmarshal([]byte(test.jsonIn), &got); err != nil {
				t.Fatal("Got error unmarshalling test input:", err)
			}
			if got.String() != test.want {
				t.Errorf("got.String() != want\ngot:  %q\nwant: %q",
					got.String(), test.want)
			}
		})
	}
}

func ExampleConfigShellCmds_AppendScript() {
	shCmds := ConfigShellCmds{}
	shCmds.AppendScript(`
		# This script will be unindented by the number of leading tabs
		# on the first line.
		if true; then
			echo "this is always printed"
		fi`,
	)
	b, _ := json.MarshalIndent(&shCmds, "", "  ")
	fmt.Println(string(b))

	// Output:
	// [
	//   "# This script will be unindented by the number of leading tabs",
	//   "# on the first line.",
	//   "if true; then",
	//   "\techo \"this is always printed\"",
	//   "fi"
	// ]
}

func TestAppendScript(t *testing.T) {
	tests := []struct {
		name     string
		script   string
		wantCmds []string
	}{
		{
			name:     "Empty",
			script:   "",
			wantCmds: nil,
		},
		{
			name:     "OnlySpaces",
			script:   " ",
			wantCmds: nil,
		},
		{
			name:     "Only newlines",
			script:   "\r\n",
			wantCmds: nil,
		},
		{
			name:     "Simple",
			script:   "echo test",
			wantCmds: []string{"echo test"},
		},
		{
			name:     "LeadingNewline",
			script:   "\necho test",
			wantCmds: []string{"echo test"},
		},
		{
			name:     "LeadingNewlineAndSpace",
			script:   "\n    echo test",
			wantCmds: []string{"echo test"},
		},
		{
			name:     "TrailingWhitespace",
			script:   "echo test  \n",
			wantCmds: []string{"echo test"},
		},
		{
			name:   "SecondLineIndent",
			script: "if true; then\n\techo test\nfi",
			wantCmds: []string{
				"if true; then",
				"\techo test",
				"fi",
			},
		},
		{
			name:   "Unindent",
			script: "\n\tif true; then\n\t\techo test\n\tfi",
			wantCmds: []string{
				"if true; then",
				"\techo test",
				"fi",
			},
		},
		{
			name:   "UnindentTooFewTabs",
			script: "\t\tif true; then\n\techo test\n\t\tfi",
			wantCmds: []string{
				"if true; then",
				"\techo test",
				"fi",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			shCmds := ConfigShellCmds{}
			shCmds.AppendScript(test.script)
			gotCmds := shCmds.Cmds
			if diff := cmp.Diff(test.wantCmds, gotCmds); diff != "" {
				t.Errorf("Got incorrect commands slice (-want +got):\n%s", diff)
			}
		})
	}
}

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
