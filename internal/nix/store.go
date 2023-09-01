package nix

import (
	"context"
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
