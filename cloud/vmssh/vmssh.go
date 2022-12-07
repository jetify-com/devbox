package vmssh

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/cloud"
	"go.jetpack.io/devbox/cloud/mutagen"
)

func SSHIfVMExists(sshArgs []string) error {
	vmAddr := vmAddressIfAny(sshArgs)

	if vmAddr != "" {
		if isActive, err := checkActiveVM(vmAddr); err != nil {
			return errors.WithStack(err)
		} else if !isActive {
			logger.log("terminating mutagen session for vm: %s", vmAddr)
			// If no vm is active, then we should terminate the running mutagen sessions
			return terminateMutagenSessions(vmAddr)
		}
	}

	return invokeSSHCmd(sshArgs)
}

func terminateMutagenSessions(vmAddr string) error {
	machineID, _, _ := strings.Cut(vmAddr, ".")

	labels := cloud.MutagenSyncLabels(machineID)
	return mutagen.Terminate(labels)
}

func checkActiveVM(vmAddr string) (bool, error) {

	cmd := exec.Command("ssh", vmAddr, "echo 'alive'")

	var bufErr, bufOut bytes.Buffer
	cmd.Stderr = &bufErr
	cmd.Stdout = &bufOut

	err := cmd.Run()
	if err != nil {
		if err.Error() == "exit status 255" {
			return false, nil
		}
		// For now, any error is deemed to indicate a VM that is no longer running.
		// We can tighten this by listening for the specific exit error code (255)
		logger.log("Error checking for Active VM: %s. Stdout: %s, Stderr: %s, cmd.Run err: %s\n",
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
	logger.log("Did not find vm address in ssh args: %v", sshArgs)
	return ""
}

func invokeSSHCmd(sshArgs []string) error {

	cmd := exec.Command("ssh", sshArgs...)
	logger.log("executing command: %s\n", cmd)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	return errors.WithStack(err)
}
