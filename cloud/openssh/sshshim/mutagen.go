package sshshim

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/cloud/mutagenbox"
	"go.jetpack.io/devbox/debug"
)

// returns true if a liveVM is found, OR sshArgs were connecting to a server that is not a devbox-VM.
// returns false iff the sshArgs were connecting to a devbox VM AND a deadVM is found.
func EnsureLiveVMOrTerminateMutagenSessions(sshArgs []string) (bool, error) {
	vmAddr := vmAddressIfAny(sshArgs)

	debug.Log("Found vmAddr: %s", vmAddr)
	if vmAddr == "" {
		// We support the no Vm scenario, in case mutagen ssh-es into another server
		// TODO savil. Revisit the no VM scenario if we can control the mutagen daemon for devbox-only
		// syncing via MUTAGEN_DATA_DIRECTORY.
		return true, nil
	}

	if isActive, err := checkActiveVM(vmAddr); err != nil {
		return false, errors.WithStack(err)
	} else if !isActive {
		debug.Log("terminating mutagen session for vm: %s", vmAddr)
		// If no vm is active, then we should terminate the running mutagen sessions
		return false, terminateMutagenSessions(vmAddr)
	}
	return true, nil
}

func terminateMutagenSessions(vmAddr string) error {
	username, hostname, found := strings.Cut(vmAddr, "@")
	if !found {
		hostname = username
	}
	machineID, _, found := strings.Cut(hostname, ".")
	if !found {
		return errors.Errorf(
			"expected to find a period (.) in hostname (%s), but did not. "+
				"For completeness, VmAddr is %s", hostname, vmAddr)
	}

	return mutagenbox.TerminateSessionsForMachine(machineID, nil /*env*/)
}

func checkActiveVM(vmAddr string) (bool, error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ssh", vmAddr, "echo 'alive'")

	var bufErr, bufOut bytes.Buffer
	cmd.Stderr = &bufErr
	cmd.Stdout = &bufOut

	err := cmd.Run()
	if err != nil {
		if e := (&exec.ExitError{}); errors.As(err, &e) && e.ExitCode() == 255 {
			debug.Log("checkActiveVM: No active VM. returning false for exit status 255")
			return false, nil
		}
		// For now, any error is deemed to indicate a VM that is no longer running.
		// We can tighten this by listening for the specific exit error code (255)
		debug.Log("Error checking for Active VM: %s. Stdout: %s, Stderr: %s, cmd.Run err: %s\n",
			vmAddr,
			bufOut.String(),
			bufErr.String(),
			err,
		)
		return false, errors.WithStack(err)
	}
	return true, nil
}

// vmAddressIfAny will seek to find the devbox-vm hostname if it exists
// in the sshArgs. If not, it returns an empty string.
func vmAddressIfAny(sshArgs []string) string {

	const devboxVMAddressSuffix = "devbox-vms.internal"
	for _, sshArg := range sshArgs {
		if strings.HasSuffix(sshArg, devboxVMAddressSuffix) {
			return sshArg
		}
	}
	debug.Log("Did not find vm address in ssh args: %v", sshArgs)
	return ""
}
