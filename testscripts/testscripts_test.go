package testrunner

import (
	"os"
	"strconv"
	"testing"

	"go.jetpack.io/devbox/testscripts/testrunner"
)

const exampleTestsEnvName = "DEVBOX_EXAMPLE_TESTS"

func TestScripts(t *testing.T) {
	testrunner.RunTestscripts(t, ".")
}

func TestMain(m *testing.M) {
	os.Exit(testrunner.Main(m))
}

// TestExamples runs testscripts on the devbox-projects in the examples folder.
func TestExamples(t *testing.T) {
	isOn, err := strconv.ParseBool(os.Getenv(exampleTestsEnvName))
	if err != nil || !isOn {
		t.Skipf("Skipping TestExamples. To enable, set %s=1.", exampleTestsEnvName)
	}

	testrunner.RunExamplesTestscripts(t, "../examples")
}
