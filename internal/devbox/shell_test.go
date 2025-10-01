// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"errors"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.jetify.com/devbox/internal/devbox/devopt"
	"go.jetify.com/devbox/internal/envir"
	"go.jetify.com/devbox/internal/shellgen"
	"go.jetify.com/devbox/internal/xdg"
)

// updateFlag overwrites golden files with the new test results.
var updateFlag = flag.Bool("update", false, "update the golden files with the test results")

func TestWriteDevboxShellrc(t *testing.T) {
	testdirs, err := filepath.Glob("testdata/shellrc/*")
	if err != nil {
		t.Fatal("Error globbing testdata:", err)
	}
	testWriteDevboxShellrc(t, testdirs)
}

func testWriteDevboxShellrc(t *testing.T, testdirs []string) {
	projectDir := "/path/to/projectDir"

	// Load up all the necessary data from each internal/nix/testdata/shellrc directory
	// into a slice of tests cases.
	tests := make([]struct {
		name            string
		env             map[string]string
		hooksFilePath   string
		shellrcPath     string
		goldShellrcPath string
		goldShellrc     []byte
	}, len(testdirs))
	var err error
	for i, path := range testdirs {
		test := &tests[i]
		test.name = filepath.Base(path)
		if b, err := os.ReadFile(filepath.Join(path, "env")); err == nil {
			test.env = envir.PairsToMap(strings.Split(string(b), "\n"))
		}

		test.hooksFilePath = shellgen.ScriptPath(projectDir, shellgen.HooksFilename)

		test.shellrcPath = filepath.Join(path, "shellrc")
		if _, err := os.Stat(test.shellrcPath); errors.Is(err, fs.ErrNotExist) {
			test.shellrcPath = ""
		}
		test.goldShellrcPath = filepath.Join(path, "shellrc.golden")
		test.goldShellrc, err = os.ReadFile(test.goldShellrcPath)
		if err != nil {
			t.Fatal("Got error reading golden file:", err)
		}
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := &DevboxShell{
				devbox:          &Devbox{projectDir: projectDir},
				env:             test.env,
				projectDir:      "/path/to/projectDir",
				userShellrcPath: test.shellrcPath,
			}
			// Set shell name based on test name for zsh tests
			if strings.Contains(test.name, "zsh") {
				s.name = shZsh
			}
			gotPath, err := s.writeDevboxShellrc()
			if err != nil {
				t.Fatal("Got writeDevboxShellrc error:", err)
			}
			gotShellrc, err := os.ReadFile(gotPath)
			if err != nil {
				t.Fatalf("Got error reading generated shellrc at %s: %v", gotPath, err)
			}

			// Overwrite the golden file if the -update flag was
			// set, and then proceed normally. The test should
			// always pass in this case.
			if *updateFlag {
				err = os.WriteFile(test.goldShellrcPath, gotShellrc, 0o666)
				if err != nil {
					t.Error("Error updating golden files:", err)
				}
			}
			goldShellrc, err := os.ReadFile(test.goldShellrcPath)
			if err != nil {
				t.Fatal("Got error reading golden file:", err)
			}
			diff := cmp.Diff(goldShellrc, gotShellrc)
			if diff != "" {
				t.Errorf(strings.TrimSpace(`
Generated shellrc != shellrc.golden (-shellrc.golden +shellrc):

	%s
If the new shellrc is correct, you can update the golden file with:

	go test -run "^%s$" -update`), diff, t.Name())
			}
		})
	}
}

func TestShellPath(t *testing.T) {
	tests := []struct {
		name     string
		envOpts  devopt.EnvOptions
		expected string
		env      map[string]string
	}{
		{
			name: "pure mode enabled",
			envOpts: devopt.EnvOptions{
				Pure: true,
			},
			expected: `^/nix/store/.*/bin/bash$`,
		},
		{
			name: "pure mode disabled",
			envOpts: devopt.EnvOptions{
				Pure: false,
			},
			env: map[string]string{
				envir.Shell: "/usr/local/bin/bash",
			},
			expected: "^/usr/local/bin/bash$",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for k, v := range test.env {
				t.Setenv(k, v)
			}
			tmpDir := t.TempDir()
			err := InitConfig(tmpDir)
			if err != nil {
				t.Fatal("Got InitConfig error:", err)
			}
			d, err := Open(&devopt.Opts{
				Dir:    tmpDir,
				Stderr: os.Stderr,
			})
			if err != nil {
				t.Fatal("Got Open error:", err)
			}
			gotPath, err := d.shellPath(test.envOpts)
			if err != nil {
				t.Fatal("Got shellPath error:", err)
			}
			matched, err := regexp.MatchString(test.expected, gotPath)
			if err != nil {
				t.Fatal("Got regexp.MatchString error:", err)
			}
			if !matched {
				t.Errorf("Expected shell path %s, but got %s", test.expected, gotPath)
			}
		})
	}
}

func TestInitShellBinaryFields(t *testing.T) {
	tests := []struct {
		name               string
		path               string
		env                map[string]string
		expectedName       name
		expectedRcPath     string
		expectedRcPathBase string
	}{
		{
			name:               "bash shell",
			path:               "/usr/bin/bash",
			expectedName:       shBash,
			expectedRcPathBase: ".bashrc",
		},
		{
			name:               "zsh shell without ZDOTDIR",
			path:               "/usr/bin/zsh",
			expectedName:       shZsh,
			expectedRcPathBase: ".zshrc",
		},
		{
			name: "zsh shell with ZDOTDIR",
			path: "/usr/bin/zsh",
			env: map[string]string{
				"ZDOTDIR": "/custom/zsh/config",
			},
			expectedName:   shZsh,
			expectedRcPath: "/custom/zsh/config/.zshrc",
		},
		{
			name:               "ksh shell",
			path:               "/usr/bin/ksh",
			expectedName:       shKsh,
			expectedRcPathBase: ".kshrc",
		},
		{
			name:           "fish shell",
			path:           "/usr/bin/fish",
			expectedName:   shFish,
			expectedRcPath: xdg.ConfigSubpath("fish/config.fish"),
		},
		{
			name:           "dash shell",
			path:           "/usr/bin/dash",
			expectedName:   shPosix,
			expectedRcPath: ".shinit",
		},
		{
			name:               "unknown shell",
			path:               "/usr/bin/unknown",
			expectedName:       shUnknown,
			expectedRcPathBase: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Set up environment variables
			for k, v := range test.env {
				t.Setenv(k, v)
			}

			shell := initShellBinaryFields(test.path)

			if shell.name != test.expectedName {
				t.Errorf("Expected shell name %v, got %v", test.expectedName, shell.name)
			}

			if test.expectedRcPath != "" {
				if shell.userShellrcPath != test.expectedRcPath {
					t.Errorf("Expected rc path %s, got %s", test.expectedRcPath, shell.userShellrcPath)
				}
			} else if test.expectedRcPathBase != "" {
				// For tests that expect a path relative to home directory,
				// check that the path ends with the expected basename
				expectedBasename := test.expectedRcPathBase
				actualBasename := filepath.Base(shell.userShellrcPath)
				if actualBasename != expectedBasename {
					t.Errorf("Expected rc path basename %s, got %s (full path: %s)", expectedBasename, actualBasename, shell.userShellrcPath)
				}
			}
		})
	}
}

func TestShellRCOverrides(t *testing.T) {
	tests := []struct {
		name         string
		shellName    name
		shellrcPath  string
		expectedEnv  map[string]string
		expectedArgs []string
	}{
		{
			name:         "bash shell",
			shellName:    shBash,
			shellrcPath:  "/tmp/devbox123/.bashrc",
			expectedArgs: []string{"--rcfile", "/tmp/devbox123/.bashrc"},
		},
		{
			name:        "zsh shell",
			shellName:   shZsh,
			shellrcPath: "/tmp/devbox123/.zshrc",
			expectedEnv: map[string]string{"ZDOTDIR": "/tmp/devbox123"},
		},
		{
			name:        "ksh shell",
			shellName:   shKsh,
			shellrcPath: "/tmp/devbox123/.kshrc",
			expectedEnv: map[string]string{"ENV": "/tmp/devbox123/.kshrc"},
		},
		{
			name:        "posix shell",
			shellName:   shPosix,
			shellrcPath: "/tmp/devbox123/.shinit",
			expectedEnv: map[string]string{"ENV": "/tmp/devbox123/.shinit"},
		},
		{
			name:         "fish shell",
			shellName:    shFish,
			shellrcPath:  "/tmp/devbox123/config.fish",
			expectedArgs: []string{"-C", ". /tmp/devbox123/config.fish"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			shell := &DevboxShell{name: test.shellName}
			extraEnv, extraArgs := shell.shellRCOverrides(test.shellrcPath)

			if test.expectedEnv != nil {
				if len(extraEnv) != len(test.expectedEnv) {
					t.Errorf("Expected %d env vars, got %d", len(test.expectedEnv), len(extraEnv))
				}
				for k, v := range test.expectedEnv {
					if extraEnv[k] != v {
						t.Errorf("Expected env var %s=%s, got %s", k, v, extraEnv[k])
					}
				}
			} else {
				if len(extraEnv) != 0 {
					t.Errorf("Expected no env vars, got %v", extraEnv)
				}
			}

			if test.expectedArgs != nil {
				if len(extraArgs) != len(test.expectedArgs) {
					t.Errorf("Expected %d args, got %d", len(test.expectedArgs), len(extraArgs))
				}
				for i, arg := range test.expectedArgs {
					if i >= len(extraArgs) || extraArgs[i] != arg {
						t.Errorf("Expected arg %d to be %s, got %s", i, arg, extraArgs[i])
					}
				}
			} else {
				if len(extraArgs) != 0 {
					t.Errorf("Expected no args, got %v", extraArgs)
				}
			}
		})
	}
}

func TestSetupShellStartupFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock zsh shell
	shell := &DevboxShell{
		name:            shZsh,
		userShellrcPath: filepath.Join(tmpDir, ".zshrc"),
	}

	// Create some test zsh startup files
	startupFiles := []string{".zshenv", ".zprofile", ".zlogin", ".zlogout", ".zimrc"}
	for _, filename := range startupFiles {
		filePath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filePath, []byte("# Test content for "+filename), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create a temporary directory for shell settings
	shellSettingsDir := t.TempDir()

	// Call setupShellStartupFiles
	shell.setupShellStartupFiles(shellSettingsDir)

	// Check that all startup files were created in the shell settings directory
	for _, filename := range startupFiles {
		expectedPath := filepath.Join(shellSettingsDir, filename)
		_, err := os.Stat(expectedPath)
		if err != nil {
			t.Errorf("Expected startup file %s to be created, but got error: %v", filename, err)
		}

		// Check that the file contains the expected template content
		content, err := os.ReadFile(expectedPath)
		if err != nil {
			t.Errorf("Failed to read created file %s: %v", filename, err)
			continue
		}

		contentStr := string(content)
		expectedOldPath := filepath.Join(tmpDir, filename)
		if !strings.Contains(contentStr, expectedOldPath) {
			t.Errorf("Expected file %s to contain path %s, but content was: %s", filename, expectedOldPath, contentStr)
		}

		if !strings.Contains(contentStr, "OLD_ZDOTDIR") {
			t.Errorf("Expected file %s to contain ZDOTDIR handling, but content was: %s", filename, contentStr)
		}
	}
}
func TestWriteDevboxShellrcBash(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test bash rc file
	bashrcPath := filepath.Join(tmpDir, ".bashrc")
	bashrcContent := "# Test bash configuration\nexport TEST_VAR=value"
	err := os.WriteFile(bashrcPath, []byte(bashrcContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test .bashrc: %v", err)
	}

	// Create a mock devbox
	devbox := &Devbox{projectDir: "/test/project"}

	// Create a bash shell
	shell := &DevboxShell{
		devbox:          devbox,
		name:            shBash,
		userShellrcPath: bashrcPath,
		projectDir:      "/test/project",
		env:             map[string]string{"TEST_ENV": "test_value"},
	}

	// Write the devbox shellrc
	shellrcPath, err := shell.writeDevboxShellrc()
	if err != nil {
		t.Fatalf("Failed to write devbox shellrc: %v", err)
	}

	// Read and verify the content
	content, err := os.ReadFile(shellrcPath)
	if err != nil {
		t.Fatalf("Failed to read generated shellrc: %v", err)
	}

	contentStr := string(content)

	// Check that it does NOT contain zsh-specific ZDOTDIR handling
	if strings.Contains(contentStr, "OLD_ZDOTDIR") {
		t.Error("Expected shellrc to NOT contain ZDOTDIR handling for bash")
	}

	// Check that it sources the original .bashrc
	if !strings.Contains(contentStr, bashrcPath) {
		t.Error("Expected shellrc to source the original .bashrc file")
	}
}

func TestWriteDevboxShellrcWithZDOTDIR(t *testing.T) {
	tmpDir := t.TempDir()

	// Set up ZDOTDIR environment
	t.Setenv("ZDOTDIR", tmpDir)

	// Create a test zsh rc file in the custom ZDOTDIR
	customZshrcPath := filepath.Join(tmpDir, ".zshrc")
	zshrcContent := "# Custom zsh configuration\nexport CUSTOM_VAR=value"
	err := os.WriteFile(customZshrcPath, []byte(zshrcContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test .zshrc: %v", err)
	}

	// Create a mock devbox
	devbox := &Devbox{projectDir: "/test/project"}

	// Create a zsh shell - this should pick up the ZDOTDIR
	shell := initShellBinaryFields("/usr/bin/zsh")
	shell.devbox = devbox
	shell.projectDir = "/test/project"

	if shell.userShellrcPath != customZshrcPath {
		t.Error("Expected shellrc path to respect ZDOTDIR")
	}

	// Write the devbox shellrc
	shellrcPath, err := shell.writeDevboxShellrc()
	if err != nil {
		t.Fatalf("Failed to write devbox shellrc: %v", err)
	}

	// Read and verify the content
	content, err := os.ReadFile(shellrcPath)
	if err != nil {
		t.Fatalf("Failed to read generated shellrc: %v", err)
	}

	contentStr := string(content)
	// Check that it contains zsh-specific ZDOTDIR handling
	if !strings.Contains(contentStr, "OLD_ZDOTDIR") {
		t.Error("Expected shellrc to contain ZDOTDIR handling for zsh")
	}

	// Check that it sources the custom .zshrc
	if !strings.Contains(contentStr, customZshrcPath) {
		t.Error("Expected shellrc to source the custom .zshrc file")
	}
}
