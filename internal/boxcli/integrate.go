// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/zealic/go2node"
	"go.jetpack.io/devbox"
)

type integrateCmdFlags struct {
	config configFlags
}

func integrateCmd() *cobra.Command {
	flags := integrateCmdFlags{}
	command := &cobra.Command{
		Use:     "integrate",
		Short:   "integrate with ide",
		Args:    cobra.MaximumNArgs(1),
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			return integrateCmdFunc(cmd, args[0], flags)
		},
	}

	flags.config.register(command)
	return command
}

type parentMessage struct {
	ConfigDir string `json:"configDir"`
}

func integrateCmdFunc(cmd *cobra.Command, ide string, flags integrateCmdFlags) error {

	if ide == "vscode" {
		// Setup process communication with node as parent
		channel, err := go2node.RunAsNodeChild()
		if err != nil {
			panic(err)
		}

		// Get config dir as a message from parent process
		msg, err := channel.Read()
		if err != nil {
			panic(err)
		}
		// Parse node process' message
		var message parentMessage
		json.Unmarshal(msg.Message, &message)
		fmt.Println(message.ConfigDir) /* todo remove */

		// todo: add error handling - send message to parent process
		box, err := devbox.Open(message.ConfigDir, cmd.OutOrStdout())
		if err != nil {
			return errors.WithStack(err)
		}
		// Get env variables of a devbox shell
		envVars, err := box.PrintEnvVars(cmd.Context())
		if err != nil {
			return errors.WithStack(err)
		}
		fmt.Println("=====")
		fmt.Println(envVars)

		// Send message to parent process to terminate
		err = channel.Write(&go2node.NodeMessage{
			Message: []byte(`{"status": "finished"}`),
		})
		if err != nil {
			panic(err)
		}
		time.Sleep(2 * time.Second)
		// Open vscode with devbox shell environment
		cmnd := exec.Command("code", "-n", message.ConfigDir)
		cmnd.Env = append(cmnd.Env, envVars...)
		err = cmnd.Run()
		if err != nil {
			fmt.Println("=====")
			fmt.Println(err.Error())
			fmt.Println("=====")
			return errors.WithStack(err)
		}
	}
	return nil
}
