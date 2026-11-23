package testrunner

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"go.jetify.com/devbox/testscripts/testrunner"
)

// When true, tests that `devbox run run_test` succeeds on every project (i.e. having devbox.json)
// found in examples/.. and testscripts/..
const runProjectTests = "DEVBOX_RUN_PROJECT_TESTS"

// When true, runs the testscripts next to the package definitions in pkgs/,
// which install real toolchains from their vendors.
const runPkgDefTests = "DEVBOX_RUN_PKGDEF_TESTS"

func TestScripts(t *testing.T) {
	// To run a specific test, say, testscripts/foo/bar.test.text, then run
	// go test ./testscripts -run TestScripts/bar
	testrunner.RunTestscripts(t, ".")
}

func TestMain(m *testing.M) {
	testrunner.Main(m)
}

// TestExamples runs testscripts on the devbox-projects in the examples folder.
func TestExamples(t *testing.T) {
	isOn, err := strconv.ParseBool(os.Getenv(runProjectTests))
	if err != nil || !isOn {
		t.Skipf("Skipping TestExamples. To enable, set %s=1.", runProjectTests)
	}

	// To run a specific test, say, examples/foo/bar, then run
	// go test ./testscripts -run TestExamples/foo_bar_run_test
	testrunner.RunDevboxTestscripts(t, "../examples")
}

// TestScriptsWithProjects runs testscripts on the devbox-projects in the testscripts folder.
func TestScriptsWithProjects(t *testing.T) {
	isOn, err := strconv.ParseBool(os.Getenv(runProjectTests))
	if err != nil || !isOn {
		t.Skipf("Skipping TestScriptsWithProjects. To enable, set %s=1.", runProjectTests)
	}

	testrunner.RunDevboxTestscripts(t, ".")
}

// TestPackageDefinitions installs every package definition in pkgs/ and
// switches between two of its versions. It downloads several gigabytes, so
// it only runs when DEVBOX_RUN_PKGDEF_TESTS is set.
//
// To run one package, say pkgs/nodejs, run
// go test ./testscripts -run TestPackageDefinitions/nodejs
func TestPackageDefinitions(t *testing.T) {
	isOn, err := strconv.ParseBool(os.Getenv(runPkgDefTests))
	if err != nil || !isOn {
		t.Skipf("Skipping TestPackageDefinitions. To enable, set %s=1.", runPkgDefTests)
	}
	testrunner.RunPackageDefinitionTestscripts(t, "../pkgs")
}

// TestPackageDefinitionScriptsExist runs on every test run (unlike
// TestPackageDefinitions) and fails when a package definition in pkgs/ has
// no testscript.
func TestPackageDefinitionScriptsExist(t *testing.T) {
	testrunner.CheckPackageDefinitionTestscripts(t, "../pkgs")
}

// TestScriptFileNames fails when a file looks like a testscript but doesn't end
// in .test.txt, which is the only suffix RunTestscripts picks up. Such a file
// never runs, so a typo in its name silently drops its coverage (this happened
// to lockfile/nopaths.txt and add/add_insecure.tst.txt).
func TestScriptFileNames(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".txt" || strings.HasSuffix(path, ".test.txt") {
			return nil
		}
		// testrunner/ holds the script that runs example projects.
		if strings.HasPrefix(filepath.ToSlash(path), "testrunner/") {
			return nil
		}
		t.Errorf("%s isn't run by the test suite: rename it to end in .test.txt", path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
