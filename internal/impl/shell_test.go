package impl

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
			test.env = pairsToMap(strings.Split(string(b), "\n"))
		}

		test.hooksFilePath = scriptPath(projectDir, hooksFilename)

		test.shellrcPath = filepath.Join(path, "shellrc")
		if _, err := os.Stat(test.shellrcPath); errors.Is(err, os.ErrNotExist) {
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
				env:             test.env,
				projectDir:      "path/to/projectDir",
				userShellrcPath: test.shellrcPath,
				hooksFilePath:   test.hooksFilePath,
				profileDir:      "./.devbox/profile",
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

func TestCleanEnvPath(t *testing.T) {
	tests := []struct {
		name    string
		inPath  string
		outPath string
	}{
		{
			name:    "NoEmptyPaths",
			inPath:  "/usr/local/bin::",
			outPath: "/usr/local/bin",
		},
		{
			name:    "NoRelativePaths",
			inPath:  "/usr/local/bin:/usr/bin:../test:/bin:/usr/sbin:/sbin:.:..",
			outPath: "/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := JoinPathLists(test.inPath)
			if got != test.outPath {
				t.Errorf("Got incorrect cleaned PATH.\ngot:  %s\nwant: %s", got, test.outPath)
			}
		})
	}
}
