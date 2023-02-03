package testscripts

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rogpeppe/go-internal/testscript"
	"github.com/stretchr/testify/require"
	"go.jetpack.io/devbox/internal/boxcli"
	"go.jetpack.io/devbox/internal/xdg"
)

func TestScripts(t *testing.T) {
	// List of directories with test scripts (files ending with .test.txt).
	dirs := globDirs("./**/*.test.txt")
	require.NotEmpty(t, dirs, "no test scripts found")

	// Loop through all the directories and run all tests scripts (files ending
	// in .test.txt)
	for _, dir := range dirs {
		t.Run(dir, func(t *testing.T) {
			testscript.Run(t, getTestscriptParams(dir))
		})
	}
}

func TestMain(m *testing.M) {
	commands := map[string]func() int{
		"devbox": func() int {
			// Call the devbox CLI directly:
			return boxcli.Execute(context.Background(), os.Args[1:])
		},
	}
	os.Exit(testscript.RunMain(m, commands))
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

func getTestscriptParams(dir string) testscript.Params {
	return testscript.Params{
		Dir:                 dir,
		RequireExplicitExec: true,
		TestWork:            false, // Set to true if you're trying to debug a test.
		Setup: func(env *testscript.Env) error {
			// Ensure path is empty so that we rely only on the PATH set by devbox
			// itself.
			// The one entry we need to keep is the /bin directory in the testing directory.
			// That directory is setup by the testing framework itself, and it's what allows
			// us to call our own custom "devbox" command.
			oldPath := env.Getenv("PATH")
			newPath := strings.Split(oldPath, ":")[0]
			env.Setenv("PATH", newPath)

			// Both devbox itself and nix occasionally create some files in
			// XDG_CACHE_HOME (which defaults to ~/.cache). For purposes of this
			// test set it to a location within the test's working directory:
			cacheHome := filepath.Join(env.WorkDir, ".cache")
			env.Setenv("XDG_CACHE_HOME", cacheHome)
			err := os.MkdirAll(cacheHome, 0755) // Ensure dir exists.
			if err != nil {
				return err
			}

			// There is one directory we do want to share across tests: nix's cache.
			// Without it tests are very slow, and nix would end up re-downloading
			// nixpkgs every time.
			// Here we create a shared location for nix's cache, and symlink from
			// the test's working directory.
			err = os.MkdirAll(xdg.CacheSubpath("devbox-tests/nix"), 0755) // Ensure dir exists.
			if err != nil {
				return err
			}
			err = os.Symlink(xdg.CacheSubpath("devbox-tests/nix"), filepath.Join(cacheHome, "nix"))
			if err != nil {
				return err
			}

			// Enable new `devbox run` so we can use it in tests. This is temporary,
			// and should be removed once we enable this feature flag.
			env.Setenv("DEVBOX_FEATURE_STRICT_RUN", "1")
			return nil
		},
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			// Usage: env.path.len <number>
			// Checks that the PATH environment variable has the expected number of entries.
			"env.path.len": func(script *testscript.TestScript, neg bool, args []string) {
				if len(args) != 1 {
					script.Fatalf("usage: env.path.len N")
				}
				expectedN, err := strconv.Atoi(args[0])
				script.Check(err)

				path := script.Getenv("PATH")
				actualN := len(strings.Split(path, ":"))
				if neg {
					if actualN == expectedN {
						script.Fatalf("path length is %d, expected != %d", actualN, expectedN)
					}
				} else {
					if actualN != expectedN {
						script.Fatalf("path length is %d, expected %d", actualN, expectedN)
					}
				}
			},

			// Usage: json.superset superset.json subset.json
			// Checks that the JSON in superset.json contains all the keys and values
			// present in subset.json.
			"json.superset": func(script *testscript.TestScript, neg bool, args []string) {
				if len(args) != 2 {
					script.Fatalf("usage: json.superset superset.json subset.json")
				}

				if neg {
					script.Fatalf("json.superset does not support negation")
				}

				data1 := script.ReadFile(args[0])
				tree1 := map[string]interface{}{}
				err := json.Unmarshal([]byte(data1), &tree1)
				script.Check(err)

				data2 := script.ReadFile(args[1])
				tree2 := map[string]interface{}{}
				err = json.Unmarshal([]byte(data2), &tree2)
				script.Check(err)

				for expectedKey, expectedValue := range tree2 {
					if actualValue, ok := tree1[expectedKey]; ok {
						if !reflect.DeepEqual(actualValue, expectedValue) {
							script.Fatalf("key '%s': expected '%v', got '%v'", expectedKey, expectedValue, actualValue)
						}
					} else {
						script.Fatalf("key '%s' not found, expected value '%v'", expectedKey, expectedValue)
					}
				}

			},
		},
	}
}
