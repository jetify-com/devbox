// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/zealic/go2node"
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

func integrateCmdFunc(cmnd *cobra.Command, ide string, flags integrateCmdFlags) error {
	// box, err := devbox.Open(flags.config.path, cmd.OutOrStdout())
	// if err != nil {
	// 	return errors.WithStack(err)
	// }
	channel, err := go2node.RunAsNodeChild()
	if err != nil {
		panic(err)
	}

	// Golang will output: {"hello":"child"}
	msg, err := channel.Read()
	if err != nil {
		panic(err)
	}
	fmt.Println(string(msg.Message))

	// Node will output: {"hello":'parent'}
	err = channel.Write(&go2node.NodeMessage{
		Message: []byte(`{"status": "finished"}`),
	})
	if err != nil {
		panic(err)
	}
	// wait till parent process exited
	// time.Sleep(2 * time.Second)
	cmd := exec.Command("code", "/Users/mohsenansari/code/jetpack/go.jetpack.io/examples/vscode/vscodetest/")
	// cmd := exec.Command("code", "/Users/mohsenansari/code/jetpack/go.jetpack.io/examples/")
	cmd.Env = append(cmd.Env, "moo=CoWsAyMoO")
	err = cmd.Run()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
	// return box.Info(pkg, flags.markdown)
}
