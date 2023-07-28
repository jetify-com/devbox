package nix

import (
	"path/filepath"
	"strings"
)

func StorePath(hash, name, version string) string {
	storeDirParts := []string{hash, name}
	if version != "" {
		storeDirParts = append(storeDirParts, version)
	}
	storeDir := strings.Join(storeDirParts, "-")
	return filepath.Join("/nix/store", storeDir)
}
