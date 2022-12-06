// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
	"go.jetpack.io/devbox/cloud"
)

type cloudShellCmdFlags struct {
	config configFlags
}

func CloudCmd() *cobra.Command {
	command := &cobra.Command{
		Use:    "cloud",
		Short:  "Remote development environments on the cloud",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	command.AddCommand(cloudShellCmd())
	command.AddCommand(cloudSshCmd())
	return command
}

func cloudShellCmd() *cobra.Command {
	flags := cloudShellCmdFlags{}

	command := &cobra.Command{
		Use:   "shell",
		Short: "Shell into a cloud environment that matches your local devbox environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCloudShellCmd(&flags)
		},
	}

	flags.config.register(command)
	return command
}

func runCloudShellCmd(flags *cloudShellCmdFlags) error {
	box, err := devbox.Open(flags.config.path, os.Stdout)
	if err != nil {
		return errors.WithStack(err)
	}
	return cloud.Shell(box.ConfigDir())
}

func cloudSshCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "ssh",
		Short: "shim for ssh",
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			logFile, err := logFile()
			if err != nil {
				return err
			}
			return runCloudSshCmd(logFile)
		},
	}

	return command
}

func runCloudSshCmd(w io.Writer) error {
	sshArgs := os.Args[3:]
	cmd := exec.Command("ssh", sshArgs...)
	fmt.Fprintf(w, "executing command: %s\n", cmd)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	//return nil
	err := cmd.Run()
	return errors.WithStack(err)
}

func logFile() (io.Writer, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	dirPath := filepath.Join(home, ".config/devbox/log")
	if err = os.MkdirAll(dirPath, 0700); err != nil {
		return nil, errors.WithStack(err)
	}

	file, err := os.OpenFile(
		filepath.Join(dirPath, "devbox_cloud_ssh.log"),
		os.O_RDWR|os.O_CREATE|os.O_TRUNC,
		0700,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return file, nil
}
