// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package envir

const (
	DevboxCache = "DEVBOX_CACHE"
	// DevboxConfig sets the default value for the --config flag, i.e. the path
	// to the directory (or devbox.json file) of the devbox project to use. This
	// is convenient for setting the config path in environments where passing
	// the flag is awkward, such as a Dockerfile.
	DevboxConfig  = "DEVBOX_CONFIG"
	DevboxGateway = "DEVBOX_GATEWAY"
	// DevboxNixBinaryCache overrides the default Nix binary cache that Devbox
	// queries for prebuilt package outputs (https://cache.nixos.org). This is
	// useful in network-restricted environments where the public cache is
	// unreachable and an internal mirror/proxy must be used instead. The value
	// should be a substituter URL serving the same store paths (e.g. an
	// Artifactory/Nexus generic remote that mirrors cache.nixos.org).
	DevboxNixBinaryCache = "DEVBOX_NIX_BINARY_CACHE"
	// DevboxLatestVersion is the latest version available of the devbox CLI binary.
	// NOTE: it should NOT start with v (like 0.4.8)
	DevboxLatestVersion  = "DEVBOX_LATEST_VERSION"
	DevboxRegion         = "DEVBOX_REGION"
	DevboxSearchHost     = "DEVBOX_SEARCH_HOST"
	DevboxShellEnabled   = "DEVBOX_SHELL_ENABLED"
	DevboxShellStartTime = "DEVBOX_SHELL_START_TIME"
	DevboxVM             = "DEVBOX_VM"

	LauncherVersion = "LAUNCHER_VERSION"
	LauncherPath    = "LAUNCHER_PATH"

	GitHubUsername = "GITHUB_USER_NAME"
	SSHTTY         = "SSH_TTY"

	XDGDataHome   = "XDG_DATA_HOME"
	XDGConfigHome = "XDG_CONFIG_HOME"
	XDGCacheHome  = "XDG_CACHE_HOME"
	XDGStateHome  = "XDG_STATE_HOME"
)

// system
const (
	Env   = "ENV"
	Home  = "HOME"
	Path  = "PATH"
	Shell = "SHELL"
	User  = "USER"
)
