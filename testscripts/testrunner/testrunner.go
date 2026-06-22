package testrunner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rogpeppe/go-internal/testscript"
	"github.com/stretchr/testify/require"
	"go.jetify.com/devbox/internal/boxcli"
)

func Main(m *testing.M) {
	commands := map[string]func(){
		"devbox": func() {
			// Call the devbox CLI directly:
			os.Exit(boxcli.Execute(context.Background(), os.Args[1:]))
		},
		"print": func() { // Not 'echo' because we don't expand variables
			fmt.Println(strings.Join(os.Args[1:], " "))
		},
	}
	testscript.Main(m, commands)
}

func RunTestscripts(t *testing.T, testscriptsDir string) {
	globPattern := filepath.Join(testscriptsDir, "**/*.test.txt")
	scripts := globScripts(globPattern)
	require.NotEmpty(t, scripts, "no test scripts found")

	shard := shardFromEnv(t)

	// Run each test script (a file ending in .test.txt) in its own
	// testscript.Run call so that we can shard at the granularity of an
	// individual script. The scripts still run as parallel subtests.
	for i, script := range scripts {
		if !shard.includes(i) {
			continue
		}
		params := getTestscriptParams(filepath.Dir(script))
		// Pass the single script explicitly rather than its directory so that
		// sharding partitions scripts, not whole directories.
		params.Dir = ""
		params.Files = []string{script}
		testscript.Run(t, params)
	}
}

// globScripts returns the test script files matching pattern, sorted for a
// deterministic order (so sharding is stable across runners). The testrunner
// dir is skipped: it holds the generic testscript used for projects in the
// examples/ directory, which is run separately (see RunDevboxTestscripts).
func globScripts(pattern string) []string {
	scripts, err := doublestar.FilepathGlob(pattern)
	if err != nil {
		return nil
	}

	filtered := scripts[:0]
	for _, script := range scripts {
		if filepath.Base(filepath.Dir(script)) == "testrunner" {
			continue
		}
		filtered = append(filtered, script)
	}
	sort.Strings(filtered)
	return filtered
}

// shard partitions the test scripts across CI runners. The testscripts are
// bound by per-runner nix work (downloading package closures and evaluating
// flakes), which does not parallelize well within a single runner, so we split
// the scripts across several runners instead.
type shard struct {
	index int // 0-based index of this runner
	total int // total number of runners
}

// shardFromEnv reads the shard configuration from the environment. When
// DEVBOX_TEST_SHARD_TOTAL is unset (or 1), all scripts run on a single runner.
// Otherwise each runner sets DEVBOX_TEST_SHARD_INDEX (0-based) and runs only the
// scripts assigned to it.
func shardFromEnv(t *testing.T) shard {
	total := 1
	if v := os.Getenv("DEVBOX_TEST_SHARD_TOTAL"); v != "" {
		n, err := strconv.Atoi(v)
		require.NoError(t, err, "invalid DEVBOX_TEST_SHARD_TOTAL=%q", v)
		require.Positive(t, n, "DEVBOX_TEST_SHARD_TOTAL must be positive")
		total = n
	}

	index := 0
	if v := os.Getenv("DEVBOX_TEST_SHARD_INDEX"); v != "" {
		n, err := strconv.Atoi(v)
		require.NoError(t, err, "invalid DEVBOX_TEST_SHARD_INDEX=%q", v)
		index = n
	}
	require.GreaterOrEqual(t, index, 0, "DEVBOX_TEST_SHARD_INDEX must be >= 0")
	require.Less(t, index, total, "DEVBOX_TEST_SHARD_INDEX must be < DEVBOX_TEST_SHARD_TOTAL")
	return shard{index: index, total: total}
}

// includes reports whether the item at position i (in a deterministic ordering)
// belongs to this shard. Round-robin assignment keeps heavy scripts spread
// across shards rather than clustered on one runner.
func (s shard) includes(i int) bool {
	return i%s.total == s.index
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
