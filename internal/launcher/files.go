// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package launcher

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"go.jetify.com/devbox/internal/xdg"
)

// The files below match the bash launcher's, because older devbox versions
// read and write them too.

// currentVersionFile holds the devbox version to run when DEVBOX_USE_VERSION
// isn't set. Its modification time is when the launcher last checked for a
// new version.
func currentVersionFile() string { return xdg.CacheSubpath("devbox/current-version") }

// latestVersionFile holds the latest devbox version from the last check.
func latestVersionFile() string { return xdg.CacheSubpath("devbox/latest-version") }

// channelFile, if it exists, holds the URL to get the latest version from
// instead of the stable channel.
func channelFile() string { return xdg.ConfigSubpath("devbox/channel") }

// autoUpdateFile turns on switching to new versions automatically, unless
// it's empty or holds "0" or "false".
func autoUpdateFile() string { return xdg.ConfigSubpath("devbox/auto_update") }

// installDir is where the launcher downloads devbox versions to. Unlike the
// files above, it's always in $HOME/.cache, even when XDG_CACHE_HOME is set.
func installDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cache", "devbox", "bin")
}

// BinaryPath returns where the launcher installs devbox version.
func BinaryPath(version string) string {
	return filepath.Join(installDir(), normalizeVersion(version)+"_"+platform(), "devbox")
}

// InstalledBinary returns the path to the devbox binary for version if it's
// installed, or an empty string otherwise.
func InstalledBinary(version string) string {
	if installDir() == "" || !validVersion(version) {
		return ""
	}
	bin := BinaryPath(version)
	if !isExecutable(bin) {
		return ""
	}
	return bin
}

// platform returns the platform part of release archive names, which is also
// the suffix of each version's directory in the install dir.
func platform() string {
	arch := runtime.GOARCH
	if arch == "arm" {
		// Releases only build 32-bit ARM for ARMv7, and name it like
		// `uname -m` does.
		arch = "armv7l"
	}
	return runtime.GOOS + "_" + arch
}

func isExecutable(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular() && fi.Mode().Perm()&0o111 != 0
}

// versionRegexp matches devbox release versions such as 0.18.4, v0.18.4, and
// 0.0.0-edge.2026-10-25. It never matches path separators, so a version is
// safe to use in file paths and URLs.
var versionRegexp = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(-?[0-9A-Za-z][0-9A-Za-z.-]*)?$`)

func validVersion(version string) bool {
	return versionRegexp.MatchString(version)
}

// normalizeVersion removes the "v" prefix, because release tags and archive
// names don't have one.
func normalizeVersion(version string) string {
	return strings.TrimPrefix(version, "v")
}

func sameVersion(a, b string) bool {
	return normalizeVersion(a) == normalizeVersion(b)
}

// readVersion returns the version in path, or an empty string if the file
// doesn't exist or doesn't hold a valid version.
func readVersion(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	version := strings.TrimSpace(string(data))
	if !validVersion(version) {
		return ""
	}
	return normalizeVersion(version)
}

// SetCurrentVersion makes version the devbox version that runs when
// DEVBOX_USE_VERSION isn't set.
func SetCurrentVersion(version string) error {
	return writeVersion(currentVersionFile(), version)
}

// SetLatestVersion saves version as the latest devbox version.
func SetLatestVersion(version string) error {
	return writeVersion(latestVersionFile(), version)
}

// writeVersion writes version to path atomically, so that a launcher that
// runs at the same time never reads part of it.
func writeVersion(path, version string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(normalizeVersion(version) + "\n"); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// readConfig returns the trimmed contents of path, or an empty string if it
// doesn't exist.
func readConfig(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func autoUpdate() bool {
	value := strings.ToLower(readConfig(autoUpdateFile()))
	return value != "" && value != "0" && value != "false"
}
