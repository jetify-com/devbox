package devbox

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.jetpack.io/devbox/planner/plansdk"
)

func TestDevbox(t *testing.T) {
	t.Setenv("TMPDIR", "/tmp")
	testPaths, err := doublestar.FilepathGlob("./testdata/**/devbox.json")
	assert.NoError(t, err, "Reading testdata/ should not fail")

	examplePaths, err := doublestar.FilepathGlob("./examples/**/devbox.json")
	assert.NoError(t, err, "Reading examples/ should not fail")

	testPaths = append(testPaths, examplePaths...)
	assert.Greater(t, len(testPaths), 0, "testdata/ and examples/ should contain at least 1 test")

	for _, testPath := range testPaths {
		testExample(t, testPath)
	}
}

func testExample(t *testing.T, testPath string) {

	currentDir, err := os.Getwd()
	require.New(t).NoError(err)

	baseDir := filepath.Dir(testPath)
	t.Run(baseDir, func(t *testing.T) {
		assert := assert.New(t)
		goldenFile := filepath.Join(baseDir, "plan.json")
		hasGoldenFile := fileExists(goldenFile)

		box, err := Open(baseDir, os.Stdout)
		assert.NoErrorf(err, "%s should be a valid devbox project", baseDir)

		// Just for tests, we make srcDir be a relative path so that the paths in plan.json
		// of various test cases have relative paths. Absolute paths are a no-go because they'd
		// be of the form `/Users/savil/...`, which are not generalized and cannot be checked in.
		box.srcDir, err = filepath.Rel(currentDir, box.srcDir)
		assert.NoErrorf(err, "expect to construct relative path from %s relative to base %s", box.srcDir, currentDir)

		plan, err := box.BuildPlan()
		buildErrorExpectedFile := filepath.Join(baseDir, "build_error_expected")
		hasBuildErrorExpectedFile := fileExists(buildErrorExpectedFile)
		if hasBuildErrorExpectedFile {
			assert.NotNil(err)
			// Since build error is expected, skip the rest of the test
			return
		}
		assert.NoError(err, "devbox plan should not fail")

		err = box.Generate()
		assert.NoError(err, "devbox generate should not fail")

		if !hasGoldenFile {
			assert.NotEmpty(plan.DevPackages, "the plan should have dev packages")
			return
		}

		data, err := os.ReadFile(goldenFile)
		assert.NoError(err, "plan.json should be readable")

		expected := &plansdk.Plan{}
		err = json.Unmarshal(data, &expected)
		assert.NoError(err, "plan.json should parse correctly")

		assertPlansMatch(t, expected, plan)
	})
}

func assertPlansMatch(t *testing.T, expected *plansdk.Plan, actual *plansdk.Plan) {
	assert := assert.New(t)

	assert.ElementsMatch(expected.DevPackages, actual.DevPackages, "DevPackages should match")
	assert.ElementsMatch(expected.RuntimePackages, actual.RuntimePackages, "RuntimePackages should match")
	assert.Equal(expected.InstallStage.GetCommand(), actual.InstallStage.GetCommand(), "Install stage should match")
	assert.Equal(expected.BuildStage.GetCommand(), actual.BuildStage.GetCommand(), "Build stage should match")
	assert.Equal(expected.StartStage.GetCommand(), actual.StartStage.GetCommand(), "Start stage should match")
	// Check that input files are the same for all stages.
	// Depending on where the test command is invoked, the input file paths can be different.
	// We will compare the file name only.
	assert.ElementsMatch(
		expected.InstallStage.GetInputFiles(),
		getFileNames(actual.InstallStage.GetInputFiles()),
		"InstallStage.InputFiles should match",
	)
	assert.ElementsMatch(
		expected.BuildStage.GetInputFiles(),
		getFileNames(actual.BuildStage.GetInputFiles()),
		"BuildStage.InputFiles should match",
	)
	assert.ElementsMatch(
		expected.StartStage.GetInputFiles(),
		actual.StartStage.GetInputFiles(),
		"StartStage.InputFiles should match",
	)

	assert.ElementsMatch(expected.Definitions, actual.Definitions, "Definitions should match")
	assert.Equal(expected.ShellInitHook, actual.ShellInitHook, "ShellInitHook should match")
	if expected.GeneratedFiles != nil {
		assert.Equal(expected.GeneratedFiles, actual.GeneratedFiles, "GeneratedFiles should match")
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func getFileNames(paths []string) []string {
	names := []string{}
	for _, path := range paths {
		if path == "." {
			names = append(names, path)
		} else {
			names = append(names, filepath.Base(path))
		}
	}

	return names
}
