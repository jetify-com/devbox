package telemetry

import (
	"github.com/denisbrodbeck/machineid"

	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/devbox/internal/env"
)

var DeviceID string

const (
	AppDevbox  = "devbox"
	AppSSHShim = "devbox-sshshim"
)

func init() {
	// TODO(gcurtis): clean this up so that Sentry and Segment use the same
	// start/stop functions.
	if env.NotTrack() || build.TelemetryKey == "" {
		return
	}
	enabled = true

	const salt = "64ee464f-9450-4b14-8d9c-014c0012ac1a"
	DeviceID, _ = machineid.ProtectedID(salt) // Ensure machine id is hashed and non-identifiable
}

var enabled bool

func Enabled() bool {
	return enabled
}
