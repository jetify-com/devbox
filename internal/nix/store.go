package nix

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
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
	cmd := commandContext(ctx, "path-info", installable, "--json")
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
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// parseStorePathFromInstallableOutput parses the output of `nix store path-from-installable --json`
// This function is decomposed out of StorePathFromInstallable to make it testable.
func parseStorePathFromInstallableOutput(installable string, output []byte) (string, error) {
	var o map[string]any
	if err := json.Unmarshal(output, &o); err != nil {
		return "", err
	}
	if len(o) > 1 {
		return "", fmt.Errorf("Found multiple store paths for installable: %s", installable)
	}
	for storePath := range o {
		return storePath, nil
	}
	return "", fmt.Errorf("Did not find store path for installable: %s", installable)
}
