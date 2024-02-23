package nix

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"go.jetpack.io/devbox/internal/debug"
	"golang.org/x/exp/maps"
)

func StorePathFromHashPart(ctx context.Context, hash, storeAddr string) (string, error) {
	cmd := commandContext(ctx, "store", "path-from-hash-part", "--store", storeAddr, hash)
	resultBytes, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(resultBytes)), nil
}

func StorePathsFromInstallable(ctx context.Context, installable string) ([]string, error) {
	// --impure for NIXPKGS_ALLOW_UNFREE
	cmd := commandContext(ctx, "path-info", installable, "--json", "--impure")
	cmd.Env = allowUnfreeEnv(os.Environ())
	debug.Log("Running cmd %s", cmd)
	resultBytes, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseStorePathFromInstallableOutput(installable, resultBytes)
}

// StorePathAreInStore returns true if the store path is in the store
// It relies on `nix store ls` to check if the store path is in the store
func StorePathsAreInStore(ctx context.Context, storePaths []string) (bool, error) {
	for _, storePath := range storePaths {
		cmd := commandContext(ctx, "store", "ls", storePath)
		debug.Log("Running cmd %s", cmd)
		if err := cmd.Run(); err != nil {
			if exitErr := (&exec.ExitError{}); errors.As(err, &exitErr) {
				return false, nil
			}
			return false, err
		}
	}
	return true, nil
}

// parseStorePathFromInstallableOutput parses the output of `nix store path-from-installable --json`
// This function is decomposed out of StorePathFromInstallable to make it testable.
func parseStorePathFromInstallableOutput(installable string, output []byte) ([]string, error) {
	// Newer nix versions (like 2.20)
	var out1 map[string]any
	if err := json.Unmarshal(output, &out1); err == nil {
		return maps.Keys(out1), nil
	}

	// Older nix versions (like 2.17)
	var out2 []struct {
		Path  string `json:"path"`
		Valid bool   `json:"valid"`
	}
	if err := json.Unmarshal(output, &out2); err == nil {
		res := []string{}
		for _, outValue := range out2 {
			res = append(res, outValue.Path)
		}
		return res, nil
	}

	return nil, fmt.Errorf("failed to parse store path from installable (%s) output: %s", installable, output)
}
