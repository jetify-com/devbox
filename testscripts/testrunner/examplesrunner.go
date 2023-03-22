package testrunner

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/rogpeppe/go-internal/testscript"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/impl"
)

// RunExamplesTestscripts generates testscripts for each example devbox-project.
func RunExamplesTestscripts(t *testing.T, examplesDir string) {
	wd, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}

	examplesDir = filepath.Join(wd, examplesDir)
	err = filepath.WalkDir(examplesDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !entry.IsDir() {
			return nil
		}

		// skip directories that do not have a devbox.json defined
		configPath := filepath.Join(path, "devbox.json")
		if _, err := os.Stat(configPath); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}

		// skip configs that do not have a run_test defined
		config, err := impl.ReadConfig(configPath)
		if err != nil {
			return err
		}
		if _, ok := config.Shell.Scripts["run_test"]; !ok {
			t.Logf("skipping config due to missing run_test at: %s\n", path)
			return nil
		}

		// TODO savil. Resolve these.
		skipList := []string{
			// These fail
			"csharp", "fsharp", "elixir", "haskell", "python", "django", "drupal", "rails",
			// jekyll passes but opens up a dialog for "approving httpd to accept incoming network connections"
			"jekyll",
		}
		for _, toSkip := range skipList {
			if strings.Contains(path, toSkip) {
				t.Logf("skipping due to skipList (%s), config at: %s\n", toSkip, path)
				return nil
			}
		}

		// TODO run in parallel
		t.Logf("running testscript for example: %s\n", path)
		runSingleExampleTestscript(t, examplesDir, path)
		return nil
	})
	if err != nil {
		t.Error(err)
	}

}

func runSingleExampleTestscript(t *testing.T, examplesDir, projectDir string) {
	testscriptDir, err := generateTestscript(examplesDir, projectDir)
	if err != nil {
		t.Error(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(testscriptDir) })

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
}

// generateTestscript will create a temp-directory and place the generic
// testscript file (.test.txt) for all examples devbox-projects in it.
// Unless there was an error, it returns the directory containing the testscript file
// and the caller is responsible for cleaning up the directory.
func generateTestscript(examplesDir, projectDir string) (testscriptDir string, err error) {
	defer func() {
		// cleanup the temp-dir if there was any error
		if err != nil {
			os.RemoveAll(testscriptDir)
		}
	}()

	testPath, err := filepath.Rel(examplesDir, projectDir)
	if err != nil {
		return "", errors.WithStack(err)
	}

	// scriptName is the generic script file used for all devbox projects
	const scriptName = "run_test.test.txt"

	// scriptNameForProject prefixes the project's path (with underscores) to the scriptName
	// so that the golang testing.T provides nice readable names for the test run
	// for each Example devbox-project.
	scriptNameForProject := fmt.Sprintf(
		"%s_%s",
		strings.ReplaceAll(testPath, "/", "_"),
		scriptName,
	)

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
	runTestScriptPath := filepath.Join(wd, "testrunner", scriptName)
	debug.Log("copying run_test.test.txt from %s to %s\n", runTestScriptPath, testscriptDir)
	// Using os's cp command for expediency.
	err = exec.Command("cp", runTestScriptPath, testscriptDir+"/"+scriptNameForProject).Run()
	return testscriptDir, errors.WithStack(err)
}
