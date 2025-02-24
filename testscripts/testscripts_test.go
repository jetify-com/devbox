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

func TestScripts(t *testing.T) {
	// To run a specific test, say, testscripts/foo/bar.test.text, then run
	// go test ./testscripts -run TestScripts/bar
	testrunner.RunTestscripts(t, ".")
}

func TestMain(m *testing.M) {
	os.Exit(testrunner.Main(m))
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
