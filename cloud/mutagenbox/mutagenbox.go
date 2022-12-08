package mutagenbox

import (
	"go.jetpack.io/devbox/cloud/mutagen"
)

// TerminateForMachine is a devbox-specific API that calls the generic mutagen terminate API.
func TerminateForMachine(machineID string, envVars map[string]string) error {
	labels := MutagenSyncLabels(machineID)
	return mutagen.Terminate(envVars, labels)
}

// Ideally, this should be in the cloud package but it leads to a compile cycle.
func MutagenSyncLabels(machineID string) map[string]string {
	return map[string]string{
		"devbox-vm": machineID,
	}
}
