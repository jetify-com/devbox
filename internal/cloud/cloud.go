// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cloud

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cloud/fly"
	"go.jetpack.io/devbox/internal/cloud/mutagen"
	"go.jetpack.io/devbox/internal/cloud/mutagenbox"
	"go.jetpack.io/devbox/internal/cloud/openssh"
	"go.jetpack.io/devbox/internal/cloud/openssh/sshshim"
	"go.jetpack.io/devbox/internal/cloud/stepper"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/telemetry"
)

func Shell(w io.Writer, projectDir string, githubUsername string) error {
	c := color.New(color.FgMagenta).Add(color.Bold)
	c.Fprintln(w, "Devbox Cloud")
	fmt.Fprintln(w, "Remote development environments powered by Nix")
	fmt.Fprint(w, "\n")

	username, vmHostname := parseVMEnvVar()
	// The flag for githubUsername overrides any env-var, since flags are a more
	// explicit action compared to an env-var which could be latently present.
	if githubUsername != "" {
		username = githubUsername
	}
	if username == "" {
		var err error
		username, err = getGithubUsername()
		if err != nil {
			return err
		}
	}
	debug.Log("username: %s", username)

	// Record the start time for telemetry, now that we are done with prompting
	// for github username.
	telemetryShellStartTime := time.Now()

	sshClient := openssh.Client{
		Username: username,
		Addr:     "gateway.devbox.sh",
	}
	// When developing we can use this env variable to point
	// to a different gateway
	var err error
	if envGateway := os.Getenv("DEVBOX_GATEWAY"); envGateway != "" {
		sshClient.Addr = envGateway
		err = openssh.SetupInsecureDebug(envGateway)
	} else {
		err = openssh.SetupDevbox()
	}
	if err != nil {
		return err
	}
	if err := sshshim.Setup(); err != nil {
		return err
	}

	if vmHostname == "" {
		stepVM := stepper.Start("Creating a virtual machine on the cloud...")
		// Inspect the ssh ControlPath to check for existing connections
		vmHostname = vmHostnameFromSSHControlPath()
		if vmHostname != "" {
			debug.Log("Using vmHostname from ssh socket: %v", vmHostname)
			stepVM.Success("Detected existing virtual machine")
		} else {
			var region string
			vmHostname, region, err = getVirtualMachine(sshClient)
			if err != nil {
				return err
			}
			stepVM.Success("Created a virtual machine in %s", fly.RegionName(region))

			// We save the username to local file only after we get a successful response
			// from the gateway, because the gateway will verify that the user's SSH keys
			// match their claimed username from github.
			err = openssh.SaveGithubUsernameToLocalFile(username)
			if err != nil {
				debug.Log("Failed to save username: %v", err)
			}
		}
	}
	debug.Log("vm_hostname: %s", vmHostname)

	s2 := stepper.Start("Starting file syncing...")
	err = syncFiles(username, vmHostname, projectDir)
	if err != nil {
		s2.Fail("Starting file syncing [FAILED]")
		return err
	}
	s2.Success("File syncing started")

	s3 := stepper.Start("Connecting to virtual machine...")
	time.Sleep(1 * time.Second)
	s3.Stop("Connecting to virtual machine")
	fmt.Fprint(w, "\n")

	return shell(username, vmHostname, projectDir, telemetryShellStartTime)
}

func PortForward(local, remote string) (string, error) {
	vmHostname := vmHostnameFromSSHControlPath()
	if vmHostname == "" {
		return "", usererr.New("No VM found. Please run `devbox cloud shell` first.")
	}
	return mutagenbox.ForwardCreate(vmHostname, local, remote)
}

func PortForwardTerminateAll() error {
	return mutagenbox.ForwardTerminateAll()
}

func PortForwardList() ([]string, error) {
	return mutagenbox.ForwardList()
}

func getGithubUsername() (string, error) {

	username, err := openssh.GithubUsernameFromLocalFile()
	if err != nil || username == "" {
		if err != nil {
			debug.Log("failed to get auth.Username. Error: %v", err)
		}

		username, err = queryGithubUsername()
		if err == nil && username != "" {
			debug.Log("Username from ssh -T git@github.com: %s", username)
		} else {
			// The query for Github username is best effort, and if it fails to resolve
			// we fallback to prompting the user, and suggesting the local computer username.
			username, err = promptUsername()
			if err != nil {
				return "", err
			}
		}
	} else {
		debug.Log("Username from locally-cached file: %s", username)
	}
	return username, nil
}

func promptUsername() (string, error) {
	username := ""
	prompt := &survey.Input{
		Message: "What is your github username?",
		Default: os.Getenv("USER"),
	}
	err := survey.AskOne(prompt, &username, survey.WithValidator(survey.Required))
	if err != nil {
		return "", errors.WithStack(err)
	}
	debug.Log("Username from prompting user: %s", username)
	return username, nil
}

type vm struct {
	JumpHost     string `json:"jump_host"`
	JumpHostPort int    `json:"jump_host_port"`
	VMHost       string `json:"vm_host"`
	VMHostPort   int    `json:"vm_host_port"`
	VMRegion     string `json:"vm_region"`
	VMPublicKey  string `json:"vm_public_key"`
	VMPrivateKey string `json:"vm_private_key"`
}

func (vm vm) redact() *vm {
	vm.VMPrivateKey = "***"
	return &vm
}

func getVirtualMachine(client openssh.Client) (vmHost string, region string, err error) {
	sshOut, err := client.Exec("auth")
	if err != nil {
		return "", "", errors.Wrapf(err, "error requesting VM")
	}
	resp := &vm{}
	if err := json.Unmarshal(sshOut, resp); err != nil {
		return "", "", errors.Wrapf(err, "error unmarshaling gateway response %q", sshOut)
	}
	if redacted, err := json.MarshalIndent(resp.redact(), "\t", "  "); err == nil {
		debug.Log("got gateway response:\n\t%s", redacted)
	}
	if resp.VMPrivateKey != "" {
		err = openssh.AddVMKey(resp.VMHost, resp.VMPrivateKey)
		if err != nil {
			return "", "", errors.Wrapf(err, "error adding new VM key")
		}
	}
	return resp.VMHost, resp.VMRegion, nil
}

func syncFiles(username, hostname, projectDir string) error {

	relProjectPathInVM, err := relativeProjectPathInVM(projectDir)
	if err != nil {
		return err
	}
	absPathInVM := absoluteProjectPathInVM(username, relProjectPathInVM)
	debug.Log("absPathInVM: %s", absPathInVM)

	err = copyConfigFileToVM(hostname, username, projectDir, absPathInVM)
	if err != nil {
		return err
	}

	env, err := mutagenbox.DefaultEnv()
	if err != nil {
		return err
	}

	ignorePaths, err := gitIgnorePaths(projectDir)
	if err != nil {
		return err
	}

	// TODO: instead of id, have the server return the machine's name and use that
	// here to. It'll make things easier to debug.
	machineID, _, _ := strings.Cut(hostname, ".")
	mutagenSessionName := mutagen.SanitizeSessionName(fmt.Sprintf("devbox-%s-%s", machineID,
		hyphenatePath(relProjectPathInVM)))

	_, err = mutagen.Sync(&mutagen.SessionSpec{
		// If multiple projects can sync to the same machine, we need the name to also include
		// the project's id.
		Name:        mutagenSessionName,
		AlphaPath:   projectDir,
		BetaAddress: fmt.Sprintf("%s@%s", username, hostname),
		// It's important that the beta path is a "clean" directory that will contain *only*
		// the projects files. If we pick a pre-existing directories with other files, those
		// files will be synced back to the local directory (due to two-way-sync) and pollute
		// the user's local project
		BetaPath: absPathInVM,
		EnvVars:  env,
		Ignore: mutagen.SessionIgnore{
			VCS:   true,
			Paths: ignorePaths,
		},
		SyncMode: "two-way-resolved",
		Labels:   mutagenbox.DefaultSyncLabels(machineID),
	})
	if err != nil {
		return err
	}
	time.Sleep(1 * time.Second)

	// In a background routine, update the sync status in the cloud VM
	go updateSyncStatus(mutagenSessionName, username, hostname, relProjectPathInVM)
	return nil
}

// updateSyncStatus updates the starship prompt.
//
// wait for the mutagen session's status to change to "watching", and update the remote VM
// when the initial project sync completes and then exit.
func updateSyncStatus(mutagenSessionName, username, hostname, relProjectPathInVM string) {

	status := "disconnected"

	// Ensure the destination directory exists
	destServer := fmt.Sprintf("%s@%s", username, hostname)
	destDir := fmt.Sprintf("/home/%s/.config/devbox/starship/%s", username, hyphenatePath(filepath.Base(relProjectPathInVM)))
	remoteCmd := fmt.Sprintf("mkdir -p %s", destDir)
	cmd := exec.Command("ssh", destServer, remoteCmd)
	err := cmd.Run()
	debug.Log("mkdir starship mutagen_status command: %s with error: %s", cmd, err)

	// Set an initial status
	displayableStatus := "initial sync"
	remoteCmd = fmt.Sprintf("echo %s > %s/mutagen_status.txt", displayableStatus, destDir)
	cmd = exec.Command("ssh", destServer, remoteCmd)
	err = cmd.Run()
	debug.Log("scp starship.toml with command: %s and error: %s", cmd, err)
	time.Sleep(5 * time.Second)

	debug.Log("Starting check for file sync status")
	for status != "watching" {
		var err error
		status, err = getSyncStatus(mutagenSessionName)
		if err != nil {
			debug.Log("ERROR: getSyncStatus error is %s", err)
			return
		}
		debug.Log("checking file sync status: %s", status)

		if status == "watching" {
			displayableStatus = "\"watching for changes\""
		}

		remoteCmd = fmt.Sprintf("echo %s > %s/mutagen_status.txt", displayableStatus, destDir)
		cmd = exec.Command("ssh", destServer, remoteCmd)
		err = cmd.Run()
		debug.Log("scp starship.toml with command: %s and error: %s", cmd, err)
		time.Sleep(5 * time.Second)
	}
}

func getSyncStatus(mutagenSessionName string) (string, error) {
	env, err := mutagenbox.DefaultEnv()
	if err != nil {
		return "", errors.WithStack(err)
	}
	sessions, err := mutagen.List(env, mutagenSessionName)
	if err != nil {
		return "", errors.WithStack(err)
	}
	if len(sessions) == 0 {
		return "", errors.WithStack(err)
	}
	return sessions[0].Status, nil
}

func copyConfigFileToVM(hostname, username, projectDir, pathInVM string) error {

	// Ensure the devbox-project's directory exists in the VM
	destServer := fmt.Sprintf("%s@%s", username, hostname)
	cmd := exec.Command("ssh", destServer, "--", "mkdir", "-p", pathInVM)
	err := cmd.Run()
	debug.Log("ssh mkdir command: %s with error: %s", cmd, err)
	if err != nil {
		return errors.WithStack(err)
	}

	// Copy the config file to the devbox-project directory in the VM
	configFilePath := filepath.Join(projectDir, "devbox.json")
	destPath := fmt.Sprintf("%s:%s", destServer, pathInVM)
	cmd = exec.Command("scp", configFilePath, destPath)
	err = cmd.Run()
	debug.Log("scp devbox.json command: %s with error: %s", cmd, err)
	return errors.WithStack(err)
}

func shell(username, hostname, projectDir string, shellStartTime time.Time) error {
	projectPath, err := relativeProjectPathInVM(projectDir)
	if err != nil {
		return err
	}

	client := &openssh.Client{
		Addr:           hostname,
		PathInVM:       absoluteProjectPathInVM(username, projectPath),
		ShellStartTime: telemetry.UnixTimestampFromTime(shellStartTime),
		Username:       username,
	}
	return client.Shell()
}

// relativeProjectPathInVM refers to the project path relative to the user's
// home-directory within the VM.
//
// Ideally, we'd pass in devbox.Devbox struct and call ProjectDir but it
// makes it hard to wrap this in a test
func relativeProjectPathInVM(projectDir string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.WithStack(err)
	}

	// get absProjectDir to expand "." and so on
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return "", errors.WithStack(err)
	}
	projectDir = filepath.Clean(absProjectDir)

	if !strings.HasPrefix(projectDir, home) {
		projectDir, err = filepath.Abs(projectDir)
		if err != nil {
			return "", errors.WithStack(err)
		}
		return filepath.Join(outsideHomedirDirectory, projectDir), nil
	}

	relativeProjectDir, err := filepath.Rel(home, projectDir)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return relativeProjectDir, nil
}

const outsideHomedirDirectory = "outside-homedir-code"

func absoluteProjectPathInVM(sshUser, relativeProjectPath string) string {
	vmHomeDir := fmt.Sprintf("/home/%s", sshUser)
	if strings.HasPrefix(relativeProjectPath, outsideHomedirDirectory) {
		return fmt.Sprintf("%s/%s", vmHomeDir, relativeProjectPath)
	}
	return fmt.Sprintf("%s/%s/", vmHomeDir, relativeProjectPath)
}

func parseVMEnvVar() (username string, vmHostname string) {
	vmEnvVar := os.Getenv("DEVBOX_VM")
	if vmEnvVar == "" {
		return "", ""
	}
	parts := strings.Split(vmEnvVar, "@")

	// DEVBOX_VM = <hostname>
	if len(parts) == 1 {
		vmHostname = parts[0]
		return
	}

	// DEVBOX_VM = <username>@<hostname>
	username = parts[0]
	vmHostname = parts[1]
	return
}

// Proof of concept: look for a gitignore file in the current directory.
// To harden this, we must:
//  1. Look for .gitignore file in each ancestor directory of projectDir, and include
//     any rules that apply to projectDir contents.
//  2. Look for .gitignore file in each child directory of projectDir and transform the
//     rules to be relative to projectDir.
func gitIgnorePaths(projectDir string) ([]string, error) {

	// We must always ignore .devbox folder. It can contain information that
	// is platform-specific, and so we should not sync it to the cloud-shell.
	// Platform-specific info includes nix profile links to the nix store,
	// and in the future, versions of specific packages in the flakes.lock file.
	result := []string{".devbox"}

	fpath := filepath.Join(projectDir, ".gitignore")
	if _, err := os.Stat(fpath); err != nil {
		if os.IsNotExist(err) {
			return result, nil
		} else {
			return nil, errors.WithStack(err)
		}
	}

	contents, err := os.ReadFile(fpath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	for _, line := range strings.Split(string(contents), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#") && line != "" {
			result = append(result, line)
		}
	}

	return result, nil
}

func vmHostnameFromSSHControlPath() string {
	for _, socket := range openssh.DevboxControlSockets() {
		if strings.HasSuffix(socket.Host, "vm.devbox-vms.internal") {
			return socket.Host
		}
	}
	// empty string means that aren't any active VM connections
	return ""
}

func hyphenatePath(path string) string {
	return strings.ReplaceAll(path, "/", "-")
}
