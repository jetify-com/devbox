// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package fileutil

import (
	"os"
	"path/filepath"

	"go.jetpack.io/devbox/internal/xdg"
)

// file path
var (
	BashConfigFile = rcfilePath(".bashrc") // default: ~/.bashrc
	KshConfigFile  = rcfilePath(".kshrc")  // default: ~/.kshrc
	ZshConfigFile  = rcfilePath(".zshrc")  // default: ~/.zshrc

	CurrentVersionFile = filepath.Join(CacheDir, "current-version") // default: ~/.cache/devbox/current-version
	NixpkgsCommitFile  = filepath.Join(CacheDir, "nixpkgs.json")    // default: ~/.cache/devbox/nixpkgs.json
	MutagenBinaryFile  = xdg.CacheSubpath("mutagen/bin/mutagen")    // default: ~/.cache/mutagen/bin/mutagen

	FishConfigFile = xdg.ConfigSubpath("fish/config.fish") // default: ~/.config/fish/config.fish

	GlobalProcessComposeJSONFile = filepath.Join(GlobalDataDir, "process-compose.json") // default: ~/.local/share/devbox/global/process-compose.json

)

// dir path
var (
	CacheDir           = xdg.CacheSubpath("devbox")           // default: ~/.cache/devbox
	NixCacheForTestDir = xdg.CacheSubpath("devbox-tests/nix") // default: ~/.cache/devbox-tests/nix

	dataDir                 = xdg.DataSubpath("devbox")                         // default: ~/.local/share/devbox
	GlobalDataDir           = filepath.Join(dataDir, "global")                  // default: ~/.local/share/devbox/global
	CurrentGlobalProfileDir = filepath.Join(GlobalDataDir, "default")           // default:  ~/.local/share/devbox/global/default (will support multiple global profiles)
	GlobalNixProfileDir     = filepath.Join(CurrentGlobalProfileDir, "profile") // default:  ~/.local/share/devbox/global/default/profile
	UtilityDataDir          = filepath.Join(dataDir, "util")                    // default: ~/.local/share/devbox/util
	UtilityNixProfileDir    = filepath.Join(UtilityDataDir, "profile")          // default: ~/.local/share/devbox/util/profile
	UtilityBinaryDir        = filepath.Join(UtilityNixProfileDir, "bin")        // default: ~/.local/share/devbox/util/profile/bin

	StateDir       = xdg.StateSubpath("devbox")                            // default: ~/.local/state/devbox
	ErrorBufferDir = xdg.StateSubpath(filepath.FromSlash("devbox/sentry")) // default: ~/.local/state/devbox/sentry
)

// rcfilePath returns the absolute path for a rcfile, which is usually in the
// user's home directory. It doesn't guarantee that the file exists.
func rcfilePath(basename string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, basename)
}

func EnsureFile(path string) error {
	if IsFile(path) {
		return nil
	}
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}

func EnsureDir(dir string) (string, error) {
	return dir, os.MkdirAll(dir, 0755)
}
