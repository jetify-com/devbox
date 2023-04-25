package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"go.jetpack.io/devbox/internal/env"
)

func TestVirtenvSymlinkPath(t *testing.T) {

	// Hardcoding XDG_STATE_HOME here so we can compare the output
	// with expected values in the test cases. Using t.TempDir() would
	// result in a randomized directory each time.
	testXdgStateHome := filepath.Join("/tmp", "devbox-virt-run-test")
	err := os.MkdirAll(testXdgStateHome, 0700)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testXdgStateHome)
	t.Setenv(env.XDGStateHome, testXdgStateHome)

	testCases := []struct {
		projectDir       string
		longXdgStateHome string
		symlinkPath      string
	}{
		// Basic directory
		{
			projectDir:  "/home/user/project",
			symlinkPath: "/tmp/devbox-virt-run-test/devbox/v-90722",
		},
		// A slightly different directory to ensure the hashing works
		{
			projectDir:  "/home/user/project/foo",
			symlinkPath: "/tmp/devbox-virt-run-test/devbox/v-5d0d3",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.projectDir, func(t *testing.T) {
			result, err := virtenvSymlinkPath(testCase.projectDir)
			if err != nil {
				t.Error(err)
			}

			if result != testCase.symlinkPath {
				t.Errorf("Expected %s, got %s", testCase.symlinkPath, result)
			}
		})
	}
}
