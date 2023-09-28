// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zealic/go2node"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/impl/devopt"
)

type integrateCmdFlags struct {
	config configFlags
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
			return runIntegrateVSCodeCmd(cmd)
		},
	}
	flags.config.register(command)

	return command
}

type parentMessage struct {
	ConfigDir string `json:"configDir"`
}

func runIntegrateVSCodeCmd(cmd *cobra.Command) error {
	// Setup process communication with node as parent
	channel, err := go2node.RunAsNodeChild()
	if err != nil {
		return err
	}

	// Get config dir as a message from parent process
	msg, err := channel.Read()
	if err != nil {
		return err
	}
	// Parse node process' message
	var message parentMessage
	if err = json.Unmarshal(msg.Message, &message); err != nil {
		return err
	}

	// todo: add error handling - consider sending error message to parent process
	box, err := devbox.Open(&devopt.Opts{
		Dir:    message.ConfigDir,
		Stderr: cmd.ErrOrStderr(),
	})
	if err != nil {
		return err
	}
	// Get env variables of a devbox shell
	envVars, err := box.EnvVars(cmd.Context())
	if err != nil {
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
	err = channel.Write(&go2node.NodeMessage{
		Message: []byte(`{"status": "finished"}`),
	})
	if err != nil {
		return err
	}
	// Open vscode with devbox shell environment
	cmnd := exec.Command("code", message.ConfigDir)
	cmnd.Env = append(cmnd.Env, envVars...)
	var outb, errb bytes.Buffer
	cmnd.Stdout = &outb
	cmnd.Stderr = &errb
	err = cmnd.Run()
	if err != nil {
		debug.Log("out: %s \n err: %s", outb.String(), errb.String())
		return err
	}
	return nil
}
