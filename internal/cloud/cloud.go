// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
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
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/envir"
	"go.jetpack.io/devbox/internal/services"
	"go.jetpack.io/devbox/internal/telemetry"
	"go.jetpack.io/devbox/internal/ux/stepper"
)

func SSHSetup(username string) (*openssh.Cmd, error) {
	sshCmd := &openssh.Cmd{
		Username:        username,
		DestinationAddr: "gateway.devbox.sh",
	}
	// When developing we can use this env variable to point
	// to a different gateway
	var err error
	if envGateway := os.Getenv(envir.DevboxGateway); envGateway != "" {
		sshCmd.DestinationAddr = envGateway
		err = openssh.SetupInsecureDebug(envGateway)
	} else {
		err = openssh.SetupDevbox()
	}
	if err != nil {
		return nil, err
	}
	if err := sshshim.Setup(); err != nil {
		return nil, err
	}
	return sshCmd, nil
}

func ensureVMForUser(vmHostname string, w io.Writer, username string, sshCmd *openssh.Cmd) (string, error) {
	if vmHostname == "" {
		color.New(color.FgGreen).Fprintln(w, "Creating a virtual machine on the cloud...")
		// Inspect the ssh ControlPath to check for existing connections
		vmHostname = vmHostnameFromSSHControlPath()
		if vmHostname != "" {
			debug.Log("Using vmHostname from ssh socket: %v", vmHostname)
			color.New(color.FgGreen).Fprintln(w, "Detected existing virtual machine")
		} else {
			var region, vmUser string
			vmUser, hostname, region, err := getVirtualMachine(sshCmd)
			if err != nil {
				return "", err
			}
			if vmUser != "" {
				username = vmUser
			}
			vmHostname = hostname
			color.New(color.FgGreen).Fprintf(w, "Created a virtual machine in %s\n", fly.RegionName(region))

			// We save the username to local file only after we get a successful response
			// from the gateway, because the gateway will verify that the user's SSH keys
			// match their claimed username from GitHub.
			err = openssh.SaveGithubUsernameToLocalFile(username)
			if err != nil {
				debug.Log("Failed to save username: %v", err)
			}
		}
	}
	return vmHostname, nil
}

func Shell(ctx context.Context, w io.Writer, projectDir, githubUsername string) error {
	color.New(color.FgMagenta, color.Bold).Fprint(w, "Devbox Cloud\n")
	fmt.Fprint(w, "Remote development environments powered by Nix\n\n")
	fmt.Fprint(w, "This is an open developer preview and may have some rough edges. Please report any issues to https://github.com/jetpack-io/devbox/issues\n\n")

	username, vmHostname, telemetryShellStartTime, err := InitVM(ctx, w, projectDir, githubUsername)
	if err != nil {
		return err
	}
	// file sync and shell
	color.New(color.FgGreen).Fprintln(w, "Starting file syncing...")
	err = syncFiles(username, vmHostname, projectDir)
	if err != nil {
		color.New(color.FgRed).Fprintln(w, "Starting file syncing [FAILED]")
		return err
	}
	color.New(color.FgGreen).Fprintln(w, "File syncing started")

	s3 := stepper.Start(w, "Connecting to virtual machine...")
	time.Sleep(1 * time.Second)
	s3.Stop("Connecting to virtual machine")
	fmt.Fprint(w, "\n")

	hostID := strings.Split(vmHostname, ".")[0]
	if err = AutoPortForward(ctx, w, projectDir, hostID); err != nil {
		return err
	}

	return shell(username, vmHostname, projectDir, telemetryShellStartTime)
}

// Temporary function to create a vm and print vmHostname to be used by devbox extension
func InitVM(
	ctx context.Context,
	w io.Writer,
	projectDir string,
	githubUsername string,
) (string, string, time.Time, error) {
	var nilTime time.Time
	if err := ensureProjectDirIsNotSensitive(projectDir); err != nil {
		return "", "", nilTime, err
	}
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
			return "", "", nilTime, err
		}
	}
	debug.Log("username: %s", username)

	// Record the start time for telemetry, now that we are done with prompting
	// for GitHub username.
	telemetryShellStartTime := time.Now()
	// setup ssh config
	sshCmd, err := SSHSetup(username)
	if err != nil {
		return "", "", nilTime, err
	}

	// creating vm for user if it doesn't exist
	vmHostname, err = ensureVMForUser(vmHostname, w, username, sshCmd)
	if err != nil {
		return "", "", nilTime, err
	}
	debug.Log("vm_hostname: %s", vmHostname)

	return username, vmHostname, telemetryShellStartTime, nil
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

func AutoPortForward(ctx context.Context, w io.Writer, projectDir, hostID string) error {
	return services.ListenToChanges(ctx,
		&services.ListenerOpts{
			HostID:     hostID,
			ProjectDir: projectDir,
			Writer:     w,
			UpdateFunc: func(service *services.ServiceStatus) (*services.ServiceStatus, bool) {
				if service == nil {
					return service, false
				}
				host := vmHostnameFromSSHControlPath()
				if host == "" {
					return service, false
				}

				saveChanges := false
				if service.Running && service.Port != "" {
					localPort, err := mutagenbox.ForwardCreateIfNotExists(host, "", service.Port)
					if err != nil {
						fmt.Fprintf(w, "Failed to create port forward for %s: %v", service.Name, err)
					}
					if service.LocalPort != localPort {
						service.LocalPort = localPort
						saveChanges = true
					}
				} else if service.Port != "" {
					if err := mutagenbox.ForwardTerminateByHostPort(host, service.Port); err != nil {
						fmt.Fprintf(w, "Failed to terminate port forward for %s: %v", service.Name, err)
					}
					if service.LocalPort != "" {
						service.LocalPort = ""
						saveChanges = true
					}
				}
				return service, saveChanges
			},
		},
	)
}

func getGithubUsername() (string, error) {
	username, err := openssh.GithubUsernameFromLocalFile()
	if err == nil && username != "" {
		debug.Log("Username from locally-cached file: %s", username)
		return username, nil
	}

	if err != nil {
		debug.Log("failed to get auth.Username. Error: %v", err)
	}
	username, err = queryGithubUsername()
	if err == nil && username != "" {
		debug.Log("Username from ssh -T git@github.com: %s", username)
		return username, nil
	}

	// The query for GitHub username is best effort, and if it fails to resolve
	// we fallback to prompting the user, and suggesting the local computer username.
	if err != nil {
		debug.Log("failed to query auth.Username. Error: %v", err)
	}
	return promptUsername()
}

func promptUsername() (string, error) {
	username := ""
	prompt := &survey.Input{
		Message: "What is your github username?",
		Default: os.Getenv(envir.User),
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
	VMUsername   string `json:"vm_username"`
	VMPublicKey  string `json:"vm_public_key"`
	VMPrivateKey string `json:"vm_private_key"`
}

func (vm vm) redact() *vm {
	vm.VMPrivateKey = "***"
	return &vm
}

func getVirtualMachine(sshCmd *openssh.Cmd) (vmUser, vmHost, region string, err error) {
	sshOut, err := sshCmd.ExecRemote("auth")
	if err != nil {
		return "", "", "", errors.Wrapf(err, "error requesting VM")
	}
	resp := &vm{}
	if err := json.Unmarshal(sshOut, resp); err != nil {
		return "", "", "", errors.Wrapf(err, "error unmarshalling gateway response %q", sshOut)
	}
	if redacted, err := json.MarshalIndent(resp.redact(), "\t", "  "); err == nil {
		debug.Log("got gateway response:\n\t%s", redacted)
	}
	if resp.VMPrivateKey != "" {
		err = openssh.AddVMKey(resp.VMHost, resp.VMPrivateKey)
		if err != nil {
			return "", "", "", errors.Wrapf(err, "error adding new VM key")
		}
	}
	return resp.VMUsername, resp.VMHost, resp.VMRegion, nil
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
	destDir := fmt.Sprintf("/home/%s/.config/devbox/starship/%s", username, hyphenatePath(filepath.Base(relProjectPathInVM)))
	mkdirCmd := openssh.Command(username, hostname)
	_, err := mkdirCmd.ExecRemote(fmt.Sprintf(`mkdir -p "%s"`, destDir))
	if err != nil {
		debug.Log("error setting initial starship mutagen status: %v", err)
	}

	// Set an initial status
	displayableStatus := "initial sync"
	statusCmd := openssh.Command(username, hostname)
	_, err = statusCmd.ExecRemote(fmt.Sprintf(`echo "%s" > "%s/mutagen_status.txt"`, displayableStatus, destDir))
	if err != nil {
		debug.Log("error setting initial starship mutagen status: %v", err)
	}
	time.Sleep(5 * time.Second)

	debug.Log("Starting check for file sync status")
	for status != "watching" {
		status, err = getSyncStatus(mutagenSessionName)
		if err != nil {
			debug.Log("ERROR: getSyncStatus error is %s", err)
			return
		}
		debug.Log("checking file sync status: %s", status)

		if status == "watching" {
			displayableStatus = "\"watching for changes\""
		}

		statusCmd := openssh.Command(username, hostname)
		_, err = statusCmd.ExecRemote(fmt.Sprintf(`echo "%s" > "%s/mutagen_status.txt"`, displayableStatus, destDir))
		if err != nil {
			debug.Log("error setting initial starship mutagen status: %v", err)
		}
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
	mkdirCmd := openssh.Command(username, hostname)
	// This is the first command we run on the VM. Sometimes is takes fly.io a few seconds
	// to propagate DNS, especially if the VM is located in a different region than
	// the proxy (this can happen if the gateway is in a different region to proxy)
	// We retry a few times to avoid failing the command.
	_, err := mkdirCmd.ExecRemoteWithRetry(fmt.Sprintf(`mkdir -p "%s"`, pathInVM), 5, 4)
	if err != nil {
		debug.Log("error copying config file to VM: %v", err)
		return errors.WithStack(err)
	}

	// Copy the config file to the devbox-project directory in the VM
	destServer := fmt.Sprintf("%s@%s", username, hostname)
	configFilePath := filepath.Join(projectDir, "devbox.json")
	destPath := fmt.Sprintf("%s:%s", destServer, pathInVM)
	cmd := exec.Command("scp", configFilePath, destPath)
	err = cmd.Run()
	debug.Log("scp devbox.json command: %s with error: %s", cmd, err)
	return errors.WithStack(err)
}

func shell(username, hostname, projectDir string, shellStartTime time.Time) error {
	projectPath, err := relativeProjectPathInVM(projectDir)
	if err != nil {
		return err
	}

	cmd := &openssh.Cmd{
		DestinationAddr: hostname,
		PathInVM:        absoluteProjectPathInVM(username, projectPath),
		ShellStartTime:  telemetry.FormatShellStart(shellStartTime),
		Username:        username,
	}
	sessionErrors := newSSHSessionErrors()
	return cloudShellErrorHandler(cmd.Shell(sessionErrors), sessionErrors)
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

func parseVMEnvVar() (username, vmHostname string) {
	vmEnvVar := os.Getenv(envir.DevboxVM)
	if vmEnvVar == "" {
		return "", ""
	}
	parts := strings.Split(vmEnvVar, "@")

	// DEVBOX_VM = <hostname>
	if len(parts) == 1 {
		vmHostname = parts[0]
		return username, vmHostname
	}

	// DEVBOX_VM = <username>@<hostname>
	username = parts[0]
	vmHostname = parts[1]
	return username, vmHostname
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
		if errors.Is(err, fs.ErrNotExist) {
			return result, nil
		}
		return nil, errors.WithStack(err)
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

func ensureProjectDirIsNotSensitive(dir string) error {
	// isSensitiveDir checks if the dir is the rootdir or the user's homedir
	isSensitiveDir := func(dir string) bool {
		dir = filepath.Clean(dir)
		if dir == "/" {
			return true
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		return dir == filepath.Clean(home)
	}

	if isSensitiveDir(dir) {
		// check for a git repository in this folder before using this project config
		// (and potentially syncing all the code to devbox-cloud)
		_, err := os.Stat(filepath.Join(dir, ".git"))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return usererr.New(
					"Found a config (devbox.json) file at %s, "+
						"but since it is a sensitive directory we require it to be part of a git repository "+
						"before we sync it to devbox cloud",
					dir,
				)
			}
			return errors.WithStack(err)
		}
	}
	return nil
}
