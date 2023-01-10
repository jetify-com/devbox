package impl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

func TestDevbox(t *testing.T) {
	t.Setenv("TMPDIR", "/tmp")
	testPaths, err := doublestar.FilepathGlob("../../examples/**/devbox.json")
	assert.NoError(t, err, "Reading testdata/ should not fail")

	assert.Greater(t, len(testPaths), 0, "testdata/ and examples/ should contain at least 1 test")

	for _, testPath := range testPaths {
		testShell(t, testPath)
	}
}

func testShell(t *testing.T, testPath string) {

	currentDir, err := os.Getwd()
	require.New(t).NoError(err)

	baseDir := filepath.Dir(testPath)
	testName := fmt.Sprintf("%s_shell_plan", filepath.Base(baseDir))
	t.Run(testName, func(t *testing.T) {
		assert := assert.New(t)
		shellPlanFile := filepath.Join(baseDir, "shell_plan.json")
		hasShellPlanFile := fileExists(shellPlanFile)

		box, err := Open(baseDir, os.Stdout)
		assert.NoErrorf(err, "%s should be a valid devbox project", baseDir)

		// Just for tests, we make projectDir be a relative path so that the paths in plan.json
		// of various test cases have relative paths. Absolute paths are a no-go because they'd
		// be of the form `/Users/savil/...`, which are not generalized and cannot be checked in.
		box.projectDir, err = filepath.Rel(currentDir, box.projectDir)
		assert.NoErrorf(err, "expect to construct relative path from %s relative to base %s", box.projectDir, currentDir)

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

func assertShellPlansMatch(t *testing.T, expected *plansdk.ShellPlan, actual *plansdk.ShellPlan) {
	assert := assert.New(t)

	assert.ElementsMatch(expected.DevPackages, actual.DevPackages, "DevPackages should match")
	assert.ElementsMatch(expected.NixOverlays, actual.NixOverlays, "NixOverlays should match")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
