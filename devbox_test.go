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

func TestPlan(t *testing.T) {
	testPaths, err := doublestar.FilepathGlob("./testdata/**/devbox.json")
	assert.NoError(t, err, "Reading testdata/ should not fail")
	assert.Greater(t, len(testPaths), 0, "testdata/ should contain at least 1 test")

	for _, testPath := range testPaths {
		baseDir := filepath.Dir(testPath)
		t.Run(baseDir, func(t *testing.T) {
			assert := assert.New(t)
			goldenFile := filepath.Join(baseDir, "plan.json")
			hasGoldenFile := fileExists(goldenFile)

			box, err := Open(baseDir)
			assert.NoErrorf(err, "%s should be a valid devbox project", baseDir)
			plan := box.Plan()
			assert.NotEmpty(plan.DevPackages, "the plan should have dev packages")
			if hasGoldenFile {
				data, err := os.ReadFile(goldenFile)
				assert.NoError(err, "plan.json should be readable")

				expected := &planner.Plan{}
				err = json.Unmarshal(data, &expected)
				assert.NoError(err, "plan.json should parse correctly")

				assert.Equal(expected, plan)
			}
		})
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
