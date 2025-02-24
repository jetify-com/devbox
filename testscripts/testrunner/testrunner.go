package testrunner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rogpeppe/go-internal/testscript"
	"github.com/stretchr/testify/require"
	"go.jetify.com/devbox/internal/boxcli"
)

func Main(m *testing.M) int {
	commands := map[string]func() int{
		"devbox": func() int {
			// Call the devbox CLI directly:
			return boxcli.Execute(context.Background(), os.Args[1:])
		},
		"print": func() int { // Not 'echo' because we don't expand variables
			fmt.Println(strings.Join(os.Args[1:], " "))
			return 0
		},
	}
	return testscript.RunMain(m, commands)
}

func RunTestscripts(t *testing.T, testscriptsDir string) {
	globPattern := filepath.Join(testscriptsDir, "**/*.test.txt")
	dirs := globDirs(globPattern)
	require.NotEmpty(t, dirs, "no test scripts found")

	// Loop through all the directories and run all tests scripts (files ending
	// in .test.txt)
	for _, dir := range dirs {
		// The testrunner dir has the testscript we use for projects in examples/ directory.
		// We should skip that one since it is run separately (see RunExamplesTestscripts).
		if filepath.Base(dir) == "testrunner" {
			continue
		}

		testscript.Run(t, getTestscriptParams(dir))
	}
}

// Return directories that contain files matching the pattern.
func globDirs(pattern string) []string {
	scripts, err := doublestar.FilepathGlob(pattern)
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

// copyFileCmd enables copying files within the WORKDIR
func copyFileCmd(script *testscript.TestScript, neg bool, args []string) {
	if len(args) < 2 {
		script.Fatalf("usage: cp <from-file> <to-file>")
	}
	if neg {
		script.Fatalf("neg does not make sense for this command")
	}
	err := script.Exec("cp", script.MkAbs(args[0]), script.MkAbs(args[1]))
	script.Check(err)
}

func globCmd(script *testscript.TestScript, neg bool, args []string) {
	count := -1
	if neg {
		count = 0
	}
	if len(args) != 0 {
		after, ok := strings.CutPrefix(args[0], "-count=")
		if ok {
			var err error
			count, err = strconv.Atoi(after)
			if err != nil {
				script.Fatalf("invalid -count=: %v", err)
			}
			if count < 1 {
				script.Fatalf("invalid -count=: must be at least 1")
			}
			args = args[1:]
		}
	}
	if len(args) == 0 {
		script.Fatalf("usage: glob [-count=N] pattern")
	}

	var matches []string
	for _, a := range args {
		glob := script.MkAbs(a)
		m, err := filepath.Glob(glob)
		if err != nil {
			script.Fatalf("invalid glob pattern: %v", err)
		}
		for _, match := range m {
			script.Logf("glob %q matched: %s", glob, match)
		}
		matches = append(matches, m...)
	}

	// -1 means that no -count= was given, so we want at least 1 match.
	if count == -1 {
		if len(matches) == 0 && !neg {
			script.Fatalf("no matches for globs %q, want at least 1", strings.Join(args, " "))
		}
		return
	}
	if len(matches) != count {
		script.Fatalf("got %d matches for globs %q, want %d", len(matches), strings.Join(args, " "), count)
	}
}

func getTestscriptParams(dir string) testscript.Params {
	return testscript.Params{
		Dir:                 dir,
		RequireExplicitExec: true,
		TestWork:            false, // Set to true if you're trying to debug a test.
		Setup:               func(env *testscript.Env) error { return setupTestEnv(env) },
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"cp":                           copyFileCmd,
			"devboxjson.packages.contains": assertDevboxJSONPackagesContains,
			"devboxlock.packages.contains": assertDevboxLockPackagesContains,
			"env.path.len":                 assertPathLength,
			"glob":                         globCmd,
			"json.superset":                assertJSONSuperset,
			"path.order":                   assertPathOrder,
			"source.path":                  sourcePath,
		},
		Condition: func(cond string) (bool, error) {
			before, key, found := strings.Cut(cond, ":")
			if found && before == "env" {
				if v, ok := os.LookupEnv(key); ok {
					return strconv.ParseBool(v)
				}
				return false, nil
			}
			return false, fmt.Errorf("unknown condition: %v", cond)
		},
	}
}
