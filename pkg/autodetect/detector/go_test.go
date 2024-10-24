package detector

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoDetectorRelevance(t *testing.T) {
	tempDir := t.TempDir()
	detector := &GoDetector{Root: tempDir}

	t.Run("No go.mod file", func(t *testing.T) {
		relevance, err := detector.Relevance(tempDir)
		assert.NoError(t, err)
		assert.Equal(t, 0.0, relevance)
	})

	t.Run("With go.mod file", func(t *testing.T) {
		err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com"), 0644)
		assert.NoError(t, err)

		relevance, err := detector.Relevance(tempDir)
		assert.NoError(t, err)
		assert.Equal(t, 1.0, relevance)
	})
}

func TestGoDetectorPackages(t *testing.T) {
	tempDir := t.TempDir()
	detector := &GoDetector{Root: tempDir}

	t.Run("No go.mod file", func(t *testing.T) {
		packages, err := detector.Packages(context.Background())
		assert.Error(t, err)
		assert.Nil(t, packages)
	})

	t.Run("With go.mod file and no version", func(t *testing.T) {
		err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com"), 0644)
		assert.NoError(t, err)

		packages, err := detector.Packages(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, []string{"go@latest"}, packages)
	})

	t.Run("With go.mod file and specific version", func(t *testing.T) {
		goModContent := `
module example.com

go 1.18
`
		err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644)
		assert.NoError(t, err)

		packages, err := detector.Packages(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, []string{"go@1.18"}, packages)
	})
}

func TestParseGoVersion(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "No version",
			content:  "module example.com",
			expected: "",
		},
		{
			name: "With version",
			content: `
module example.com

go 1.18
`,
			expected: "1.18",
		},
		{
			name: "With patch version",
			content: `
module example.com

go 1.18.3
`,
			expected: "1.18.3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			version := parseGoVersion(tc.content)
			assert.Equal(t, tc.expected, version)
		})
	}
}
