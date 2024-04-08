// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zealic/go2node"
	"go.jetpack.io/devbox/internal/devbox"
	"go.jetpack.io/devbox/internal/devbox/devopt"
)

type integrateCmdFlags struct {
	config    configFlags
	debugmode bool
}

func integrateCmd() *cobra.Command {
	command := &cobra.Command{
		Use:     "integrate",
		Short:   "integrate with an IDE",
		Args:    cobra.MaximumNArgs(1),
		Hidden:  true,
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	command.AddCommand(integrateVSCodeCmd())
	return command
}

func integrateVSCodeCmd() *cobra.Command {
	flags := integrateCmdFlags{}
	command := &cobra.Command{
		Use:    "vscode",
		Hidden: true,
		Short:  "Integrate devbox environment with VSCode.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIntegrateVSCodeCmd(cmd, flags)
		},
	}
	command.Flags().BoolVar(&flags.debugmode, "debugmode", false, "enable debug outputs to a file.")
	flags.config.register(command)

	return command
}

type parentMessage struct {
	ConfigDir string `json:"configDir"`
}

func runIntegrateVSCodeCmd(cmd *cobra.Command, flags integrateCmdFlags) error {
	dbug := debugMode{
		enabled: flags.debugmode,
	}
	// Setup process communication with node as parent
	dbug.logToFile("Devbox process initiated. Setting up communication channel with VSCode process")
	channel, err := go2node.RunAsNodeChild()
	if err != nil {
		dbug.logToFile(err.Error())
		return err
	}
	// Get config dir as a message from parent process
	msg, err := channel.Read()
	if err != nil {
		dbug.logToFile(err.Error())
		return err
	}
	// Parse node process' message
	var message parentMessage
	if err = json.Unmarshal(msg.Message, &message); err != nil {
		dbug.logToFile(err.Error())
		return err
	}

	// todo: add error handling - consider sending error message to parent process
	box, err := devbox.Open(&devopt.Opts{
		Dir:    message.ConfigDir,
		Stderr: cmd.ErrOrStderr(),
	})
	if err != nil {
		dbug.logToFile(err.Error())
		return err
	}
	// Get env variables of a devbox shell
	dbug.logToFile("Computing devbox environment")
	envVars, err := box.EnvVars(cmd.Context())
	if err != nil {
		dbug.logToFile(err.Error())
		return err
	}
	envVars = slices.DeleteFunc(envVars, func(s string) bool {
		k, _, ok := strings.Cut(s, "=")
		// DEVBOX_OG_PATH_<hash> being set causes devbox global shellenv to overwrite the
		// PATH after VSCode opens and resets it to global shellenv. This causes the VSCode
		// terminal to not be able to find devbox packages after the reopen in devbox
		// environment action is called.
		return ok && (strings.HasPrefix(k, "DEVBOX_OG_PATH") || k == "HOME" || k == "NODE_CHANNEL_FD")
	})

	// Send message to parent process to terminate
	dbug.logToFile("Signaling VSCode to close")
	err = channel.Write(&go2node.NodeMessage{
		Message: []byte(`{"status": "finished"}`),
	})
	if err != nil {
		dbug.logToFile(err.Error())
		return err
	}
	// Open vscode with devbox shell environment
	cmnd := exec.Command("code", message.ConfigDir)
	cmnd.Env = append(cmnd.Env, envVars...)
	var outb, errb bytes.Buffer
	cmnd.Stdout = &outb
	cmnd.Stderr = &errb
	dbug.logToFile("Re-opening VSCode in computed devbox environment")
	err = cmnd.Run()
	if err != nil {
		dbug.logToFile(fmt.Sprintf("stdout: %s \n stderr: %s", outb.String(), errb.String()))
		dbug.logToFile(err.Error())
		return err
	}
	return nil
}

type debugMode struct {
	enabled bool
}

func (d *debugMode) logToFile(msg string) {
	// only write to file when --debugmode=true flag is passed
	if d.enabled {
		file, err := os.OpenFile(".devbox/extension.log", os.O_APPEND|os.O_WRONLY, 0o666)
		if err != nil {
			log.Fatal(err)
		}
		timestamp := time.Now().UTC().Format(time.RFC1123)
		_, err = file.WriteString("[" + timestamp + "] " + msg + "\n")
		if err != nil {
			log.Fatal(err)
		}
		if err = file.Close(); err != nil {
			log.Fatal(err)
		}
	}
}
