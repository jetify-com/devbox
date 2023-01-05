// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cloud

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/cloud/mutagen"
	"go.jetpack.io/devbox/internal/cloud/mutagenbox"
	"go.jetpack.io/devbox/internal/cloud/openssh"
	"go.jetpack.io/devbox/internal/cloud/openssh/sshshim"
	"go.jetpack.io/devbox/internal/cloud/stepper"
	"go.jetpack.io/devbox/internal/debug"
)

func Shell(configDir string) error {
	c := color.New(color.FgMagenta).Add(color.Bold)
	c.Println("Devbox Cloud")
	fmt.Println("Blazingly fast remote development that feels local")
	fmt.Print("\n")

	username, vmHostname := parseVMEnvVar()
	if username == "" {
		stepGithubUsername := stepper.Start("Detecting your Github username...")
		var err error
		username, err = queryGithubUsername()
		if err == nil && username != "" {
			stepGithubUsername.Success("Username: %s", username)
		} else {
			stepGithubUsername.Fail("Unable to resolve username")
			// The query for Github username is best effort, and if it fails to resolve
			// we fallback to prompting the user, and suggesting the local computer username.
			username = promptUsername()
		}
	}
	debug.Log("username: %s", username)

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
			vmHostname = getVirtualMachine(sshClient)
			stepVM.Success("Created virtual machine")
		}
	}
	debug.Log("vm_hostname: %s", vmHostname)

	s2 := stepper.Start("Starting file syncing...")
	err = syncFiles(username, vmHostname, configDir)
	if err != nil {
		s2.Fail("Starting file syncing [FAILED]")
		log.Fatal(err)
	}
	s2.Success("File syncing started")

	s3 := stepper.Start("Connecting to virtual machine...")
	time.Sleep(1 * time.Second)
	s3.Stop("Connecting to virtual machine")
	fmt.Print("\n")

	return shell(username, vmHostname, configDir)
}

func promptUsername() string {
	username := ""
	prompt := &survey.Input{
		Message: "What is your github username?",
		Default: os.Getenv("USER"),
	}
	debug.Log("Failed to get username from Github. Falling back to suggesting $USER: %s", prompt.Default)
	err := survey.AskOne(prompt, &username, survey.WithValidator(survey.Required))
	if err != nil {
		log.Fatal(err)
	}
	return username
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

func getVirtualMachine(client openssh.Client) string {
	sshOut, err := client.Exec("auth")
	if err != nil {
		log.Fatalln("error requesting VM:", err)
	}
	resp := &vm{}
	if err := json.Unmarshal(sshOut, resp); err != nil {
		log.Fatalf("error unmarshaling gateway response %q: %v", sshOut, err)
	}
	if redacted, err := json.MarshalIndent(resp.redact(), "\t", "  "); err == nil {
		debug.Log("got gateway response:\n\t%s", redacted)
	}
	if resp.VMPrivateKey == "" {
		return resp.VMHost
	}

	err = openssh.AddVMKey(resp.VMHost, resp.VMPrivateKey)
	if err != nil {
		log.Fatalf("error adding new VM key: %v", err)
	}
	return resp.VMHost
}

func syncFiles(username, hostname, configDir string) error {

	projectName := projectDirName(configDir)
	debug.Log("Will sync files to directory: ~/code/%s", projectName)

	err := copyConfigFileToVM(hostname, username, configDir, projectName)
	if err != nil {
		return err
	}

	env, err := mutagenbox.DefaultEnv()
	if err != nil {
		return err
	}

	ignorePaths, err := gitIgnorePaths(configDir)
	if err != nil {
		return err
	}

	// TODO: instead of id, have the server return the machine's name and use that
	// here to. It'll make things easier to debug.
	machineID, _, _ := strings.Cut(hostname, ".")
	mutagenSessionName := mutagen.SanitizeSessionName(fmt.Sprintf("devbox-%s-%s", projectName, machineID))
	_, err = mutagen.Sync(&mutagen.SessionSpec{
		// If multiple projects can sync to the same machine, we need the name to also include
		// the project's id.
		Name:        mutagenSessionName,
		AlphaPath:   configDir,
		BetaAddress: fmt.Sprintf("%s@%s", username, hostname),
		// It's important that the beta path is a "clean" directory that will contain *only*
		// the projects files. If we pick a pre-existing directories with other files, those
		// files will be synced back to the local directory (due to two-way-sync) and pollute
		// the user's local project
		BetaPath: projectPathInVM(projectName),
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
	go updateSyncStatus(mutagenSessionName, username, hostname, projectName)
	return nil
}

// updateSyncStatus updates the starship prompt.
//
// wait for the mutagen session's status to change to "watching", and update the remote VM
// when the initial project sync completes and then exit.
func updateSyncStatus(mutagenSessionName, username, hostname, projectName string) {
	status := "disconnected"

	// Ensure the destination directory exists
	destServer := fmt.Sprintf("%s@%s", username, hostname)
	destDir := fmt.Sprintf("/home/%s/.config/devbox/starship/%s", username, projectName)
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

func copyConfigFileToVM(hostname, username, configDir, projectName string) error {

	// Ensure the devbox-project's directory exists in the VM
	destServer := fmt.Sprintf("%s@%s", username, hostname)
	cmd := exec.Command("ssh", destServer, "--", "mkdir", "-p", projectPathInVM(projectName))
	err := cmd.Run()
	debug.Log("ssh mkdir command: %s with error: %s", cmd, err)
	if err != nil {
		return errors.WithStack(err)
	}

	// Copy the config file to the devbox-project directory in the VM
	configFilePath := filepath.Join(configDir, "devbox.json")
	destPath := fmt.Sprintf("%s:%s", destServer, projectPathInVM(projectName))
	cmd = exec.Command("scp", configFilePath, destPath)
	err = cmd.Run()
	debug.Log("scp devbox.json command: %s with error: %s", cmd, err)
	return errors.WithStack(err)
}

func projectPathInVM(projectName string) string {
	return fmt.Sprintf("~/code/%s/", projectName)
}

func shell(username, hostname, configDir string) error {
	client := &openssh.Client{
		Username:       username,
		Addr:           hostname,
		ProjectDirName: projectDirName(configDir),
	}
	return client.Shell()
}

const defaultProjectDirName = "devbox_project"

// Ideally, we'd pass in devbox.Devbox struct and call ConfigDir but it
// makes it hard to wrap this in a test
func projectDirName(configDir string) string {
	name := filepath.Base(configDir)
	if name == "/" || name == "." {
		return defaultProjectDirName
	}
	return name
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
//  1. Look for .gitignore file in each ancestor directory of configDir, and include
//     any rules that apply to configDir contents.
//  2. Look for .gitignore file in each child directory of configDir and transform the
//     rules to be relative to configDir.
func gitIgnorePaths(configDir string) ([]string, error) {

	// We must always ignore .devbox folder. It can contain information that
	// is platform-specific, and so we should not sync it to the cloud-shell.
	// Platform-specific info includes nix profile links to the nix store,
	// and in the future, versions of specific packages in the flakes.lock file.
	result := []string{".devbox"}

	fpath := filepath.Join(configDir, ".gitignore")
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
