package testrunner

import (
	"os"
	"strconv"
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
