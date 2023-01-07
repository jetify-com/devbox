package cloud

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProjectDirName(t *testing.T) {

	testCases := []struct {
		projectDir string
		dirPath    string
	}{
		// TODO revisit
		//{"/", defaultProjectDirName},
		//{".", defaultProjectDirName},
		{"/foo", "foo"},
		{"foo/bar", "bar"},
		{"foo/bar/", "bar"},
		{"foo/bar///", "bar"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.projectDir, func(t *testing.T) {
			assert := assert.New(t)
			path, err := projectDirPath(testCase.projectDir)
			assert.NoError(err)
			assert.Equal(testCase.dirPath, path)
		})
	}
}
