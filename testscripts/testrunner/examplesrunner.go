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
	err := filepath.WalkDir(examplesDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !entry.IsDir() {
			return nil
		}

		configPath := filepath.Join(path, "devbox.json")
		config, err := impl.ReadConfig(configPath)
		if err != nil {
			// skip directories that do not have a devbox.json defined
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		// skip configs that do not have a run_test defined
		if _, ok := config.Shell.Scripts["run_test"]; !ok {
			t.Logf("skipping config due to missing run_test at: %s\n", path)
			return nil
		}

		// TODO savil. Resolve these.
		skipList := []string{
			// elixir:
			//         ** (Mix) Could not compile dependency :ranch,
			//         ".nix-mix/elixir/1-14/rebar3 bare compile --paths $WORK/_build/dev/lib/*/ebin"
			//         command failed. Errors may have been logged above.
			//         You can recompile this dependency with "mix deps.compile ranch",
			//         update it with "mix deps.update ranch" or clean it with "mix deps.clean ranch"
			"elixir",

			// failing: https://github.com/jetpack-io/devbox/actions/runs/4504377069/jobs/7928774568
			"haskell",

			// pip: $WORK/.devbox/virtenv/python310Packages.pip/.venv/bin/activate: No such file or directory
			"pip",

			// django: $WORK/.devbox/virtenv/python310Packages.pip/.venv/bin/activate: No such file or directory
			"django",

			// drupal:
			//       sh: line 1: pkill: command not found
			//        Service "php-fpm" failed to stop. Error = exit status 127
			//        sh: line 1: pkill: command not found
			//        Service "nginx" failed to stop. Error = exit status 127
			//        mysqladmin: connect to server at 'localhost' failed
			//        error: 'Can't create UNIX socket (2)'
			"drupal",

			// rails:
			//         $WORK/.devbox/gen/scripts/run_test.sh: line 3: cd: blog: No such file or directory
			//        curl: (7) Failed to connect to localhost port 3000 after 0 ms: Couldn't connect to server
			"rails",

			// jekyll passes but opens up a dialog for "approving httpd to accept incoming network connections"
			"jekyll",

			// pipenv takes too long: 350 seconds
			"pipenv",
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
	testscriptDir, err := generateTestscript(t, examplesDir, projectDir)
	if err != nil {
		t.Error(err)
	}

	params := getTestscriptParams(testscriptDir)

	// save a reference to the original params.Setup so that we can wrap it below
	setup := params.Setup
	params.Setup = func(env *testscript.Env) error {
		// setup the devbox testscript environment
		if err := setup(env); err != nil {
			return errors.WithStack(err)
		}

		// We set a HOME env-var because:
		// 1. testscripts overrides it to /no-home, presumably to improve isolation
		// 2. but many language tools rely on a $HOME being set, and break due to 1.
		//    examples include ~/.dotnet folder and GOCACHE=$HOME/Library/Caches/go-build
		// We deliberately set this for examplesrunner since we are dealing with
		// language stacks, and not for the testrunner which has devbox unit tests.
		env.Setenv("HOME", t.TempDir())

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
// It returns the directory containing the testscript file.
func generateTestscript(t *testing.T, examplesDir, projectDir string) (string, error) {

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
	testscriptDir := t.TempDir()

	// Copy the testscript file to the temp-dir
	runTestScriptPath := filepath.Join("testrunner", scriptName)
	debug.Log("copying run_test.test.txt from %s to %s\n", runTestScriptPath, testscriptDir)
	// Using os's cp command for expediency.
	err = exec.Command("cp", runTestScriptPath, testscriptDir+"/"+scriptNameForProject).Run()
	return testscriptDir, errors.WithStack(err)
}
