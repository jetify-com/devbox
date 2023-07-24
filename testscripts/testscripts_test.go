package testrunner

import (
	"os"
	"strconv"
	"testing"

	"go.jetpack.io/devbox/testscripts/testrunner"
)

// When true, tests that `devbox run run_test` succeeds on every devbox.json
// found in examples/.. and testscripts/..
const runDevboxJSONTests = "DEVBOX_RUN_DEVBOX_JSON_TESTS"

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
	isOn, err := strconv.ParseBool(os.Getenv(runDevboxJSONTests))
	if err != nil || !isOn {
		t.Skipf("Skipping TestExamples. To enable, set %s=1.", runDevboxJSONTests)
	}

	// To run a specific test, say, examples/foo/bar, then run
	// go test ./testscripts -run TestExamples/foo_bar_run_test
	testrunner.RunDevboxTestscripts(t, "../examples")
}

// TestScriptsWithDevboxJSON runs testscripts on the devbox-projects in the testscripts folder.
func TestScriptsWithDevboxJSON(t *testing.T) {
	isOn, err := strconv.ParseBool(os.Getenv(runDevboxJSONTests))
	if err != nil || !isOn {
		t.Skipf("Skipping TestExamples. To enable, set %s=1.", runDevboxJSONTests)
	}

	testrunner.RunDevboxTestscripts(t, ".")
}
