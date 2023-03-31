package telemetry

import (
	"os"
	"strconv"

	"github.com/denisbrodbeck/machineid"
	"go.jetpack.io/devbox/internal/build"
)

var DeviceID string

const (
	AppDevbox  = "devbox"
	AppSSHShim = "devbox-sshshim"
)

func init() {
	// TODO(gcurtis): clean this up so that Sentry and Segment use the same
	// start/stop functions.
	if DoNotTrack() || build.TelemetryKey == "" {
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

func DoNotTrack() bool {
	// https://consoledonottrack.com/
	doNotTrack, _ := strconv.ParseBool(os.Getenv("DO_NOT_TRACK"))
	return doNotTrack
}
