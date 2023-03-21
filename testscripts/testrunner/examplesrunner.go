package testrunner

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/rogpeppe/go-internal/testscript"
	"go.jetpack.io/devbox/internal/debug"
)

// RunExamplesTestscripts generates testscripts for each example devbox-project.
// For now, we prototype with the "go" example project. TODO savil: generalize.
func RunExamplesTestscripts(t *testing.T, examplesDir string) {
	wd, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}

	// TODO savil. Change this to handle _all_ of the example folder's devbox-projects
	examplesDir = filepath.Join(wd, examplesDir)
	projectDir := filepath.Join(examplesDir, "development", "go", "hello-world")
	runSingleExampleTestscript(t, examplesDir, projectDir)
}

func runSingleExampleTestscript(t *testing.T, examplesDir, projectDir string) {
	testscriptDir, err := generateTestscript()
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(testscriptDir)

	testName, err := filepath.Rel(examplesDir, projectDir)
	if err != nil {
		t.Error(err)
	}

	t.Run(testName, func(t *testing.T) {
		params := getTestscriptParams(testscriptDir)

		// save a reference to the original params.Setup so that we can wrap it below
		setup := params.Setup
		params.Setup = func(env *testscript.Env) error {
			// setup the devbox testscript environment
			if err := setup(env); err != nil {
				return errors.WithStack(err)
			}

			// copy all the files and folders of the devbox-project being tested to the workdir
			debug.Log("copying projectDir: %s to env.WorkDir: %s\n", projectDir, env.WorkDir)
			// implementation detail: the period at the end of the projectDir/.
			// is important to ensure this works for both mac and linux.
			// Ref.https://dev.to/ackshaey/macos-vs-linux-the-cp-command-will-trip-you-up-2p00
			err = exec.Command("cp", "-r", projectDir+"/.", env.WorkDir).Run()
			if err != nil {
				return errors.WithStack(err)
			}

			return errors.WithStack(err)
		}

		testscript.Run(t, params)
	})
}

// generateTestscript will create a temp-directory and place the generic
// testscript file (.test.txt) for all examples devbox-projects in it.
// Unless there was an error, it returns the directory containing the testscript file
// and the caller is responsible for cleaning up the directory.
func generateTestscript() (testscriptDir string, err error) {
	defer func() {
		// cleanup the temp-dir if there was any error
		if err != nil {
			os.RemoveAll(testscriptDir)
		}
	}()

	// create a temp-dir to place the testscript file
	testscriptDir, err = os.MkdirTemp("", "example")
	if err != nil {
		return "", errors.WithStack(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		return testscriptDir, errors.WithStack(err)
	}

	// Copy the testscript file to the temp-dir
	runTestScriptPath := filepath.Join(wd, "testrunner", "run_test.test.txt")
	debug.Log("copying run_test.test.txt from %s to %s\n", runTestScriptPath, testscriptDir)
	// Using os's cp command for expediency.
	err = exec.Command("cp", runTestScriptPath, testscriptDir).Run()
	return testscriptDir, errors.WithStack(err)
}
