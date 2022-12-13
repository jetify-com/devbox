package telemetry

import (
	"os"
	"strconv"
)

func DoNotTrack() bool {
	// https://consoledonottrack.com/
	doNotTrack_, err := strconv.ParseBool(os.Getenv("DO_NOT_TRACK"))
	if err != nil {
		doNotTrack_ = false
	}
	return doNotTrack_
}
