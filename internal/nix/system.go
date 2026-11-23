// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import "runtime"

// SystemOrHost returns the nix system (see [System]) or, when nix is not
// installed and System returns "", the nix system name of the machine
// Devbox runs on, derived from GOOS and GOARCH.
//
// Code paths that must work without nix, such as package definitions, use
// this instead of System.
func SystemOrHost() string {
	if system := System(); system != "" {
		return system
	}
	return HostSystem()
}

// HostSystem returns the nix system name ("x86_64-linux", "aarch64-darwin",
// ...) for the GOOS and GOARCH Devbox was built for. It does not run nix.
func HostSystem() string {
	return hostSystem(runtime.GOOS, runtime.GOARCH)
}

func hostSystem(goos, goarch string) string {
	arch := goarch
	switch goarch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "aarch64"
	case "386":
		arch = "i686"
	}
	return arch + "-" + goos
}
