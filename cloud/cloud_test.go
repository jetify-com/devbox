package cloud

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigDirName(t *testing.T) {

	testCases := []struct {
		configDir string
		dirName   string
	}{
		{"/", defaultProjectDirName},
		{".", defaultProjectDirName},
		{"/foo", "foo"},
		{"foo/bar", "bar"},
		{"foo/bar/", "bar"},
		{"foo/bar///", "bar"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.configDir, func(t *testing.T) {
			assert := assert.New(t)
			assert.Equal(testCase.dirName, projectDirName(testCase.configDir))
		})
	}
}
