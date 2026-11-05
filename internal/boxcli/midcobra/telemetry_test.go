package midcobra

import (
	"testing"

	"go.jetify.com/devbox/internal/devbox"
	"go.jetify.com/devbox/internal/devbox/devopt"
)

func TestGetPackagesAndCommitHash(t *testing.T) {
	dir := t.TempDir()
	err := devbox.InitConfig(dir)
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}
	box, err := devbox.Open(&devopt.Opts{
		Dir: dir,
	})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	packages, commitHash := getPackagesAndCommitHash()

	// Check if the returned packages and commitHash are as expected
	if len(packages) != 0 {
		t.Errorf("Expected no packages, got %d", len(packages))
	}

	if commitHash != box.Lockfile().Stdenv().Rev {
		t.Errorf("Expected commitHash %s, got %s", box.Lockfile().Stdenv().Rev, commitHash)
	}
}
