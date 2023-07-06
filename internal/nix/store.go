package nix

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

func StorePath(hash, name, version string) string {
	storeDirParts := []string{hash, name}
	if version != "" {
		storeDirParts = append(storeDirParts, version)
	}
	storeDir := strings.Join(storeDirParts, "-")
	return filepath.Join("/nix/store", storeDir)
}

// contentAddressedRegex matches the output of `nix store make-content-addressed`.
// It is used to select the content-addressed store path (the second one in the example below).
//
// Example:
// > nix store make-content-addressed  /nix/store/r2jd6ygnmirm2g803mksqqjm4y39yi6i-git-2.33.1
// rewrote '/nix/store/r2jd6ygnmirm2g803mksqqjm4y39yi6i-git-2.33.1' to '/nix/store/ldbhlwhh39wha58rm61bkiiwm6j7211j-git-2.33.1'
var contentAddressedRegex = regexp.MustCompile(`rewrote\s'[\/a-z0-9-\.]+'\sto\s'([a-z0-9-\/\.]+)'`)

// ContentAddressedStorePath takes a store path and returns the content-addressed store path.
func ContentAddressedStorePath(storePath string) (string, error) {
	cmd := command("store", "make-content-addressed", storePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.WithStack(err)
	}

	matches := contentAddressedRegex.FindStringSubmatch(string(out))
	if len(matches) < 2 {
		return "", errors.Errorf("could not parse output of nix store make-content-addressed: %s", string(out))
	}

	return matches[1], nil
}
