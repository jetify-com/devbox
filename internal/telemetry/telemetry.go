package telemetry

import (
	"os"
	"runtime"
	"strconv"

	"github.com/denisbrodbeck/machineid"
	"go.jetpack.io/devbox/internal/fileutil"
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

func OS() string {
	os := runtime.GOOS
	// Special case for WSL, which is reported as 'linux' otherwise.
	if fileutil.Exists("/proc/sys/fs/binfmt_misc/WSLInterop") || fileutil.Exists("/run/WSL") {
		os = "wsl"
	}

	return os
}
