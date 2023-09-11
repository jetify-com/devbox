// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package build

import (
	"runtime"
	"sync"

	"go.jetpack.io/devbox/internal/fileutil"
)

// Variables in this file are set via ldflags.
var (
	IsDev      = Version == "0.0.0-dev"
	Version    = "0.0.0-dev"
	Commit     = "none"
	CommitDate = "unknown"

	// SentryDSN is injected in the build from
	// https://jetpack-io.sentry.io/settings/projects/devbox/keys/
	// It is disabled by default.
	SentryDSN = ""
	// TelemetryKey is the Segment Write Key
	// https://segment.com/docs/connections/sources/catalog/libraries/server/go/quickstart/
	// It is disabled by default.
	TelemetryKey = ""
)

// User-presentable names of operating systems supported by Devbox.
const (
	OSLinux  = "Linux"
	OSDarwin = "macOS"
	OSWSL    = "WSL"
)

var (
	osName string
	osOnce sync.Once
)

func OS() string {
	osOnce.Do(func() {
		switch runtime.GOOS {
		case "linux":
			if fileutil.Exists("/proc/sys/fs/binfmt_misc/WSLInterop") || fileutil.Exists("/run/WSL") {
				osName = OSWSL
			}
			osName = OSLinux
		case "darwin":
			osName = OSDarwin
		default:
			osName = runtime.GOOS
		}
	})
	return osName
}
