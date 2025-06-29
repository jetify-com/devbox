package midcobra

import (
	"testing"

	"github.com/spf13/cobra"
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
	// Create a mock cobra command
	cmd := &cobra.Command{
		Use: "test",
		Run: func(cmd *cobra.Command, args []string) {},
	}

	// Add a mock flag to the command
	cmd.Flags().String("config", "", "config file")
	if err := cmd.Flags().Set("config", dir); err != nil {
		t.Errorf("Expected no error, got %s", err)
	}

	// Call the function with the mock command
	packages, commitHash := getPackagesAndCommitHash(cmd)

	// Check if the returned packages and commitHash are as expected
	if len(packages) != 0 {
		t.Errorf("Expected no packages, got %d", len(packages))
	}

	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}

	if commitHash != box.Lockfile().Stdenv().Rev {
		t.Errorf("Expected commitHash %s, got %s", box.Lockfile().Stdenv().Rev, commitHash)
	}
}
