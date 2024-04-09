// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cloud

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProjectDirName(t *testing.T) {
	assertion := assert.New(t)

	homeDir, err := os.UserHomeDir()
	assertion.NoError(err)

	workingDir, err := os.Getwd()
	assertion.NoError(err)

	relWorkingDir, err := filepath.Rel(homeDir, workingDir)
	assertion.NoError(err)

	testCases := []struct {
		projectDir string
		dirPath    string
	}{
		// inside homedir
		{".", relWorkingDir},
		{filepath.Join(homeDir, "foo"), "foo"},
		{filepath.Join(homeDir, "foo/bar"), "foo/bar"},

		// non-home-dir
		{"/", filepath.Join(outsideHomedirDirectory, "/")},
		{"/foo", filepath.Join(outsideHomedirDirectory, "/foo")},
		{"/foo/bar", filepath.Join(outsideHomedirDirectory, "/foo/bar")},
		{"/foo/bar/", filepath.Join(outsideHomedirDirectory, "/foo/bar")},
		{"/foo/bar///", filepath.Join(outsideHomedirDirectory, "/foo/bar")},
	}

	for _, testCase := range testCases {
		t.Run(testCase.projectDir, func(t *testing.T) {
			assert := assert.New(t)
			path, err := relativeProjectPathInVM(testCase.projectDir)
			assert.NoError(err)
			assert.Equal(testCase.dirPath, path)
		})
	}
}
