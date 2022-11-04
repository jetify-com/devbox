package devbox

import (
	"encoding/json"
	"fmt"
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
		testShell(t, testPath)
		testBuild(t, testPath)
	}
}

func testShell(t *testing.T, testPath string) {

	currentDir, err := os.Getwd()
	require.New(t).NoError(err)

	baseDir := filepath.Dir(testPath)
	testName := fmt.Sprintf("%s_shell_plan", baseDir)
	t.Run(testName, func(t *testing.T) {
		assert := assert.New(t)
		shellPlanFile := filepath.Join(baseDir, "shell_plan.json")
		hasShellPlanFile := fileExists(shellPlanFile)

		box, err := Open(baseDir, os.Stdout)
		assert.NoErrorf(err, "%s should be a valid devbox project", baseDir)

		// Just for tests, we make srcDir be a relative path so that the paths in plan.json
		// of various test cases have relative paths. Absolute paths are a no-go because they'd
		// be of the form `/Users/savil/...`, which are not generalized and cannot be checked in.
		box.srcDir, err = filepath.Rel(currentDir, box.srcDir)
		assert.NoErrorf(err, "expect to construct relative path from %s relative to base %s", box.srcDir, currentDir)

		shellPlan, err := box.ShellPlan()
		assert.NoError(err, "devbox shell plan should not fail")

		err = box.generateShellFiles()
		assert.NoError(err, "devbox generate should not fail")

		if !hasShellPlanFile {
			assert.NotEmpty(shellPlan.DevPackages, "the plan should have dev packages")
			return
		}

		data, err := os.ReadFile(shellPlanFile)
		assert.NoError(err, "shell_plan.json should be readable")

		expected := &plansdk.ShellPlan{}
		err = json.Unmarshal(data, &expected)
		assert.NoError(err, "plan.json should parse correctly")
		assertShellPlansMatch(t, expected, shellPlan)
	})
}

func testBuild(t *testing.T, testPath string) {

	currentDir, err := os.Getwd()
	require.New(t).NoError(err)

	baseDir := filepath.Dir(testPath)
	testName := fmt.Sprintf("%s_build_plan", baseDir)
	t.Run(testName, func(t *testing.T) {
		assert := assert.New(t)
		buildPlanFile := filepath.Join(baseDir, "build_plan.json")
		hasBuildPlanFile := fileExists(buildPlanFile)

		box, err := Open(baseDir, os.Stdout)
		assert.NoErrorf(err, "%s should be a valid devbox project", baseDir)

		// Just for tests, we make srcDir be a relative path so that the paths in plan.json
		// of various test cases have relative paths. Absolute paths are a no-go because they'd
		// be of the form `/Users/savil/...`, which are not generalized and cannot be checked in.
		box.srcDir, err = filepath.Rel(currentDir, box.srcDir)
		assert.NoErrorf(err, "expect to construct relative path from %s relative to base %s", box.srcDir, currentDir)

		buildPlan, err := box.BuildPlan()
		buildErrorExpectedFile := filepath.Join(baseDir, "build_error_expected")
		hasBuildErrorExpectedFile := fileExists(buildErrorExpectedFile)
		if hasBuildErrorExpectedFile {
			assert.NotNil(err)
			// Since build error is expected, skip the rest of the test
			return
		}
		assert.NoError(err, "devbox plan should not fail")

		err = box.generateBuildFiles()
		assert.NoError(err, "devbox generate should not fail")

		if !hasBuildPlanFile {
			assert.NotEmpty(buildPlan.DevPackages, "the plan should have dev packages")
			return
		}

		data, err := os.ReadFile(buildPlanFile)
		assert.NoError(err, "plan.json should be readable")

		expected := &plansdk.BuildPlan{}
		err = json.Unmarshal(data, &expected)
		assert.NoError(err, "plan.json should parse correctly")
		assertBuildPlansMatch(t, expected, buildPlan)
	})
}

func assertShellPlansMatch(t *testing.T, expected *plansdk.ShellPlan, actual *plansdk.ShellPlan) {
	assert := assert.New(t)

	assert.ElementsMatch(expected.DevPackages, actual.DevPackages, "DevPackages should match")
	assert.ElementsMatch(expected.NixOverlays, actual.NixOverlays, "NixOverlays should match")
}

func assertBuildPlansMatch(t *testing.T, expected *plansdk.BuildPlan, actual *plansdk.BuildPlan) {
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
