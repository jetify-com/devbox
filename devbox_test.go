package devbox

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/stretchr/testify/assert"
	"go.jetpack.io/devbox/planner"
)

func TestDevboxPlan(t *testing.T) {
	testPaths, err := doublestar.FilepathGlob("./testdata/**/devbox.json")
	assert.NoError(t, err, "Reading testdata/ should not fail")

	examplePaths, err := doublestar.FilepathGlob("./examples/**/devbox.json")
	assert.NoError(t, err, "Reading examples/ should not fail")

	testPaths = append(testPaths, examplePaths...)
	assert.Greater(t, len(testPaths), 0, "testdata/ and examples/ should contain at least 1 test")

	for _, testPath := range testPaths {
		testIndividualPlan(t, testPath)
	}
}

func testIndividualPlan(t *testing.T, testPath string) {
	baseDir := filepath.Dir(testPath)
	t.Run(baseDir, func(t *testing.T) {
		assert := assert.New(t)
		goldenFile := filepath.Join(baseDir, "plan.json")
		hasGoldenFile := fileExists(goldenFile)

		box, err := Open(baseDir)
		assert.NoErrorf(err, "%s should be a valid devbox project", baseDir)
		plan := box.Plan()

		if !hasGoldenFile {
			assert.NotEmpty(plan.DevPackages, "the plan should have dev packages")
			return
		}

		data, err := os.ReadFile(goldenFile)
		assert.NoError(err, "plan.json should be readable")

		expected := &planner.Plan{}
		err = json.Unmarshal(data, &expected)
		assert.NoError(err, "plan.json should parse correctly")
		expected.Errors = nil

		assert.ElementsMatch(expected.DevPackages, plan.DevPackages, "DevPackages should match")
		assert.ElementsMatch(expected.RuntimePackages, plan.RuntimePackages, "RuntimePackages should match")
		assert.Equal(expected.InstallStage.GetCommand(), plan.InstallStage.GetCommand(), "Install stage should match")
		assert.Equal(expected.BuildStage.GetCommand(), plan.BuildStage.GetCommand(), "Build stage should match")
		assert.Equal(expected.StartStage.GetCommand(), plan.StartStage.GetCommand(), "Start stage should match")
		// Check that input files are the same for all stages.
		// Depending on where the test command is invoked, the input file paths can be different.
		// We will compare the file name only.
		assert.ElementsMatch(expected.InstallStage.GetInputFiles(), getFileNames(plan.InstallStage.GetInputFiles()), "InstallStage.InputFiles should match")
		assert.ElementsMatch(expected.BuildStage.GetInputFiles(), getFileNames(plan.BuildStage.GetInputFiles()), "BuildStage.InputFiles should match")
		assert.ElementsMatch(expected.StartStage.GetInputFiles(), getFileNames(plan.StartStage.GetInputFiles()), "StartStage.InputFiles should match")
	})
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
