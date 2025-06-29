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
