// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cloud

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/cloud/mutagen"
	"go.jetpack.io/devbox/cloud/sshclient"
	"go.jetpack.io/devbox/cloud/sshconfig"
	"go.jetpack.io/devbox/cloud/stepper"
	"go.jetpack.io/devbox/debug"
)

func Shell(box *devbox.Devbox) error {
	setupSSHConfig()

	c := color.New(color.FgMagenta).Add(color.Bold)
	c.Println("Devbox Cloud")
	fmt.Println("Blazingly fast remote development that feels local")
	fmt.Print("\n")

	username := promptUsername()
	s1 := stepper.Start("Creating a virtual machine on the cloud...")
	vmHostname := getVirtualMachine(username)
	s1.Success("Created virtual machine")

	s2 := stepper.Start("Starting file syncing...")
	err := syncFiles(username, vmHostname, box)
	if err != nil {
		s2.Fail("Starting file syncing [FAILED]")
		log.Fatal(err)
	}
	s2.Success("File syncing started")

	s3 := stepper.Start("Connecting to virtual machine...")
	time.Sleep(1 * time.Second)
	s3.Stop("Connecting to virtual machine")

	fmt.Print("\n")

	return shell(username, vmHostname)
}

func setupSSHConfig() {
	if err := sshconfig.Setup(); err != nil {
		log.Fatal(err)
	}
}

func promptUsername() string {
	username := ""
	prompt := &survey.Input{
		Message: "What is your github username?",
		Default: os.Getenv("USER"),
	}
	err := survey.AskOne(prompt, &username, survey.WithValidator(survey.Required))
	if err != nil {
		log.Fatal(err)
	}
	return username
}

type authResponse struct {
	VMHostname string `json:"vm_host"`
}

func getVirtualMachine(username string) string {
	// When developing we can use this env variable to point
	// to a different gateway
	hostname := os.Getenv("DEVBOX_GATEWAY")
	if hostname == "" {
		hostname = "gateway.devbox.sh"
	}
	client := sshclient.Client{
		Username: username,
		Hostname: hostname,
	}
	bytes, err := client.Exec("auth")
	if err != nil {
		log.Fatal(err)
	}
	resp := &authResponse{}
	err = json.Unmarshal(bytes, resp)
	if err != nil {
		log.Fatal(err)
	}

	return resp.VMHostname
}

func syncFiles(username string, hostname string, box *devbox.Devbox) error {
	projectName := projectDirName(box.ConfigDir())
	debug.Log("Will sync files to directory: ~/code/%s", projectName)

	// TODO: instead of id, have the server return the machine's name and use that
	// here to. It'll make things easier to debug.
	id, _, _ := strings.Cut(hostname, ".")
	_, err := mutagen.Sync(&mutagen.SessionSpec{
		// If multiple projects can sync to the same machine, we need the name to also include
		// the project's id.
		Name:        fmt.Sprintf("devbox-%s", id),
		AlphaPath:   box.ConfigDir(),
		BetaAddress: fmt.Sprintf("%s@%s", username, hostname),
		// It's important that the beta path is a "clean" directory that will contain *only*
		// the projects files. If we pick a pre-existing directories with other files, those
		// files will be synced back to the local directory (due to two-way-sync) and pollute
		// the user's local project
		BetaPath:  "~/Code/", // TODO Use ~/code/{projectName}
		IgnoreVCS: true,
		SyncMode:  "two-way-resolved",
	})
	if err != nil {
		return err
	}
	time.Sleep(1 * time.Second)
	return nil
}

func shell(username string, hostname string) error {
	client := &sshclient.Client{
		Username: username,
		Hostname: hostname,
	}
	return client.Shell()
}

const defaultProjectDirName = "DevboxProject"

// Ideally, we'd pass in devbox.Devbox struct and call ConfigDir but it
// makes it hard to wrap this in a test
func projectDirName(configDir string) string {
	name := filepath.Base(configDir)
	if name == "/" || name == "." {
		return defaultProjectDirName
	}
	return name
}
