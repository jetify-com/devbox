package telemetry

import (
	"os"
	"strconv"

	"github.com/denisbrodbeck/machineid"
)

func DoNotTrack() bool {
	// https://consoledonottrack.com/
	doNotTrack, err := strconv.ParseBool(os.Getenv("DO_NOT_TRACK"))
	if err != nil {
		doNotTrack = false
	}
	return doNotTrack
}

func DeviceID() string {
	salt := "64ee464f-9450-4b14-8d9c-014c0012ac1a"
	hashedID, _ := machineid.ProtectedID(salt) // Ensure machine id is hashed and non-identifiable
	return hashedID
}
