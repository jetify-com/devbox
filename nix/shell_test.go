package nix

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// update overwrites golden files with the new test results.
var update = flag.Bool("update", false, "update the golden files with the test results")

func TestWriteDevboxShellrc(t *testing.T) {
	testdirs, err := filepath.Glob("testdata/shellrc/*")
	if err != nil {
		t.Fatal("Error globbing testdata:", err)
	}

	// Load up all the necessary data from each testdata/shellrc directory
	// into a slice of tests cases.
	tests := make([]struct {
		name            string
		env             []string
		hook            string
		shellrcPath     string
		goldShellrcPath string
		goldShellrc     []byte
	}, len(testdirs))
	for i, path := range testdirs {
		test := &tests[i]
		test.name = filepath.Base(path)
		if b, err := os.ReadFile(filepath.Join(path, "env")); err == nil {
			test.env = strings.Split(string(b), "\n")
		}
		if b, err := os.ReadFile(filepath.Join(path, "hook")); err == nil {
			test.hook = string(b)
		}
		test.shellrcPath = filepath.Join(path, "shellrc")
		if _, err := os.Stat(test.shellrcPath); errors.Is(err, os.ErrNotExist) {
			test.shellrcPath = "noshellrc"
		}
		test.goldShellrcPath = filepath.Join(path, "shellrc.golden")
		test.goldShellrc, err = os.ReadFile(test.goldShellrcPath)
		if err != nil {
			t.Fatal("Got error reading golden file:", err)
		}
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotPath, err := writeDevboxShellrc(test.shellrcPath, test.hook, test.env)
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
			if *update {
				err = os.WriteFile(test.goldShellrcPath, gotShellrc, 0666)
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
