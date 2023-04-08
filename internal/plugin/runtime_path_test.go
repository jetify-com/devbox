package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVirtenvRuntimeLinkPath(t *testing.T) {

	// Intentionally not using t.TempDir() here because it results in a "long path",
	// which virtenvRuntimeLinkPath() handles differently.
	testXdgRuntimeDir := filepath.Join("/tmp", "devbox-virt-run-test")
	err := os.MkdirAll(testXdgRuntimeDir, 0700)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testXdgRuntimeDir)
	t.Setenv("XDG_RUNTIME_DIR", testXdgRuntimeDir)

	hundredCharPath := strings.Repeat("a", 100)
	longXdgRuntimeDir, err := os.MkdirTemp("", hundredCharPath)
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		projectDir        string
		longXdgRuntimeDir string
		runtimeLinkPath   string
	}{
		// Basic directory
		{
			projectDir:      "/home/user/project",
			runtimeLinkPath: "/tmp/devbox-virt-run-test/devbox/v-18242",
		},
		// A slightly different directory to ensure the hashing works
		{
			projectDir:      "/home/user/project/foo",
			runtimeLinkPath: "/tmp/devbox-virt-run-test/devbox/v-19648",
		},
		// An XDG Runtime directory that is very long, so that runtimeLinkPath is calculated by
		// falling back to /tmp/user-<uid>/devbox/v-<hash>
		{
			projectDir:        "/home/user/project",
			longXdgRuntimeDir: longXdgRuntimeDir,
			runtimeLinkPath: filepath.Join("/tmp", fmt.Sprintf("user-%d", os.Getuid()),
				"devbox/v-18242"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.projectDir, func(t *testing.T) {
			if testCase.longXdgRuntimeDir != "" {
				t.Setenv("XDG_RUNTIME_DIR", testCase.longXdgRuntimeDir)
			}
			result, err := virtenvRuntimeLinkPath(testCase.projectDir)
			if err != nil {
				t.Error(err)
			}

			if result[:len(result)-10] != testCase.runtimeLinkPath[:len(testCase.runtimeLinkPath)-10] {
				t.Errorf("Expected %s, got %s", testCase.runtimeLinkPath, result)
			}
		})
	}
}
