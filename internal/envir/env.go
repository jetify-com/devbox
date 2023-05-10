// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package envir

const (
	DevboxCache              = "DEVBOX_CACHE"
	devboxCLICloudShell      = "DEVBOX_CLI_CLOUD_SHELL"
	DevboxDebug              = "DEVBOX_DEBUG"
	DevboxFeaturePrefix      = "DEVBOX_FEATURE_"
	DevboxGateway            = "DEVBOX_GATEWAY"
	DevboxDoNotUpgradeConfig = "DEVBOX_DONT_UPGRADE_CONFIG"
	// DevboxLatestVersion is the latest version available of the devbox CLI binary.
	// NOTE: it should NOT start with v (like 0.4.8)
	DevboxLatestVersion  = "DEVBOX_LATEST_VERSION"
	DevboxOGPathPrefix   = "DEVBOX_OG_PATH_"
	DevboxRegion         = "DEVBOX_REGION"
	DevboxRunCmd         = "DEVBOX_RUN_CMD"
	DevboxSearchHost     = "DEVBOX_SEARCH_HOST"
	devboxShellEnabled   = "DEVBOX_SHELL_ENABLED"
	DevboxShellStartTime = "DEVBOX_SHELL_START_TIME"
	DevboxVM             = "DEVBOX_VM"

	LauncherVersion = "LAUNCHER_VERSION"
	LauncherPath    = "LAUNCHER_PATH"

	SSHTTY = "SSH_TTY"

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
