// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package envir

const (
	DevboxCache         = "DEVBOX_CACHE"
	DevboxFeaturePrefix = "DEVBOX_FEATURE_"
	DevboxGateway       = "DEVBOX_GATEWAY"
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
