package nix

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/fileutil"
	"go.jetpack.io/devbox/internal/xdg"
)

// ensureNixpkgsPrefetched runs the prefetch step to download the flake of the registry
func ensureNixpkgsPrefetched(w io.Writer, commit string) error {
	// Look up the cached map of commitHash:nixStoreLocation
	commitToLocation, err := nixpkgsCommitFileContents()
	if err != nil {
		return err
	}

	// Check if this nixpkgs.Commit is located in the local /nix/store
	location, isPresent := commitToLocation[commit]
	if isPresent {
		if fi, err := os.Stat(location); err == nil && fi.IsDir() {
			// The nixpkgs for this commit hash is present, so we don't need to prefetch
			return nil
		}
	}

	fmt.Fprintf(w, "Ensuring nixpkgs registry is downloaded.\n")
	cmd := exec.Command(
		"nix", "flake", "prefetch",
		FlakeNixpkgs(commit),
	)
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)
	cmd.Env = DefaultEnv()
	cmd.Stdout = w
	cmd.Stderr = cmd.Stdout
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(w, "Ensuring nixpkgs registry is downloaded: ")
		color.New(color.FgRed).Fprintf(w, "Fail\n")
		return errors.Wrapf(err, "Command: %s", cmd)
	}
	fmt.Fprintf(w, "Ensuring nixpkgs registry is downloaded: ")
	color.New(color.FgGreen).Fprintf(w, "Success\n")

	return saveToNixpkgsCommitFile(commit, commitToLocation)
}

func nixpkgsCommitFileContents() (map[string]string, error) {
	path := nixpkgsCommitFilePath()
	if !fileutil.Exists(path) {
		return map[string]string{}, nil
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	commitToLocation := map[string]string{}
	if err := json.Unmarshal(contents, &commitToLocation); err != nil {
		return nil, errors.WithStack(err)
	}
	return commitToLocation, nil
}

func saveToNixpkgsCommitFile(commit string, commitToLocation map[string]string) error {
	// Make a query to get the /nix/store path for this commit hash.
	cmd := exec.Command("nix", "flake", "prefetch", "--json",
		FlakeNixpkgs(commit),
	)
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)
	out, err := cmd.Output()
	if err != nil {
		return errors.WithStack(err)
	}

	// read the json response
	var prefetchData struct {
		StorePath string `json:"storePath"`
	}
	if err := json.Unmarshal(out, &prefetchData); err != nil {
		return errors.WithStack(err)
	}

	// Ensure the nixpkgs commit file path exists so we can write an update to it
	path := nixpkgsCommitFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil && !errors.Is(err, fs.ErrExist) {
		return errors.WithStack(err)
	}

	// write to the map, jsonify it, and write that json to the nixpkgsCommit file
	commitToLocation[commit] = prefetchData.StorePath
	serialized, err := json.Marshal(commitToLocation)
	if err != nil {
		return errors.WithStack(err)
	}

	err = os.WriteFile(path, serialized, 0644)
	return errors.WithStack(err)
}

func nixpkgsCommitFilePath() string {
	cacheDir := xdg.CacheSubpath("devbox")
	return filepath.Join(cacheDir, "nixpkgs.json")
}
