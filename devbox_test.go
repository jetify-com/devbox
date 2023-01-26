package devbox

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rogpeppe/go-internal/testscript"
	"github.com/stretchr/testify/require"
)

func TestScripts(t *testing.T) {
	// List of directories with test scripts.
	dirs := findScriptDirs()
	require.NotEmpty(t, dirs, "no test scripts found in testdata/")

	// Loop through all the directories and run all tests scripts (files ending
	// in .test.txt)
	for _, dir := range dirs {
		t.Run(dir, func(t *testing.T) {
			testscript.Run(t, getTestscriptParams(dir))
		})
	}
}

func TestMain(m *testing.M) {
	commands := map[string]func() int{}
	os.Exit(testscript.RunMain(m, commands))
}

// Find directories that contain test scripts (files ending in .test.txt)
func findScriptDirs() []string {
	scripts, err := doublestar.FilepathGlob("testdata/**/*.test.txt")
	if err != nil {
		return nil
	}

	// List of directories with test scripts.
	directories := []string{}
	dups := map[string]bool{}
	for _, script := range scripts {
		dir := filepath.Dir(script)
		if _, ok := dups[dir]; !ok {
			directories = append(directories, dir)
			dups[dir] = true
		}
	}

	return directories
}

type cmdMap map[string]func(ts *testscript.TestScript, neg bool, args []string)

func getTestscriptParams(dir string) testscript.Params {
	return testscript.Params{
		Dir:                 dir,
		RequireExplicitExec: true,
		Setup: func(env *testscript.Env) error {
			// Ensure path is empty so that we rely only on the PATH set by devbox
			// itself.
			env.Setenv("PATH", "")
			return nil
		},
		Cmds: cmdMap{},
	}
}
