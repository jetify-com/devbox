package telemetry

import (
	"os"
	"strconv"

	"github.com/denisbrodbeck/machineid"
)

var DeviceID string

const (
	AppDevbox  = "devbox"
	AppSSHShim = "devbox-sshshim"
)

func init() {

	if DoNotTrack() {
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
