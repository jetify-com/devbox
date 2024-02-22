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
)

func StorePathFromHashPart(ctx context.Context, hash, storeAddr string) (string, error) {
	cmd := commandContext(ctx, "store", "path-from-hash-part", "--store", storeAddr, hash)
	resultBytes, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(resultBytes)), nil
}

func StorePathFromInstallable(ctx context.Context, installable string) (string, error) {
	// --impure for NIXPKGS_ALLOW_UNFREE
	cmd := commandContext(ctx, "path-info", installable, "--json", "--impure")
	cmd.Env = allowUnfreeEnv(os.Environ())
	debug.Log("Running cmd %s", cmd)
	resultBytes, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return parseStorePathFromInstallableOutput(installable, resultBytes)
}

// StorePathIsInStore returns true if the store path is in the store
// It relies on `nix store ls` to check if the store path is in the store
func StorePathIsInStore(ctx context.Context, storePath string) (bool, error) {
	cmd := commandContext(ctx, "store", "ls", storePath)
	debug.Log("Running cmd %s", cmd)
	if err := cmd.Run(); err != nil {
		if exitErr := (&exec.ExitError{}); errors.As(err, &exitErr) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// parseStorePathFromInstallableOutput parses the output of `nix store path-from-installable --json`
// This function is decomposed out of StorePathFromInstallable to make it testable.
func parseStorePathFromInstallableOutput(installable string, output []byte) (string, error) {
	var out1 map[string]any
	if err := json.Unmarshal(output, &out1); err == nil {
		if len(out1) > 1 {
			return "", fmt.Errorf("found multiple store paths for installable: %s", installable)
		}
		for storePath := range out1 {
			return storePath, nil
		}
		return "", fmt.Errorf("did not find store path for installable: %s", installable)
	}

	var out2 []struct {
		Path  string `json:"path"`
		Valid bool   `json:"valid"`
	}
	if err := json.Unmarshal(output, &out2); err == nil {
		for _, outValue := range out2 {
			return outValue.Path, nil
		}
	}

	return "", fmt.Errorf("failed to parse store path from installable output: %s", output)
}
