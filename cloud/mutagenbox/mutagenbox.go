package mutagenbox

import (
	"go.jetpack.io/devbox/cloud/mutagen"
)

// TerminateSessionsForMachine is a devbox-specific API that calls the generic mutagen terminate API.
// It relies on the mutagen-sync-session's labels to identify which sessions to terminate for
// a particular machine (fly VM).
func TerminateSessionsForMachine(machineID string, env map[string]string) error {
	labels := MutagenSyncLabels(machineID)
	return mutagen.Terminate(env, labels)
}

// Ideally, this should be in the cloud package but it leads to a compile cycle.
func MutagenSyncLabels(machineID string) map[string]string {
	return map[string]string{
		"devbox-vm": machineID,
	}
}
