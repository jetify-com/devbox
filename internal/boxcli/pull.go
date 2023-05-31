// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox"
)

type pullCmdFlags struct {
	config configFlags
	force  bool
}

func pullCmd() *cobra.Command {
	flags := pullCmdFlags{}
	cmd := &cobra.Command{
		Use:     "pull <file> | <url>",
		Short:   "Pull a config from a file or URL",
		Long:    "Pull a config from a file or URL. URLs must be prefixed with 'http://' or 'https://'.",
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			return pullCmdFunc(cmd, args[0], &flags)
		},
		Args: cobra.ExactArgs(1),
	}

	cmd.Flags().BoolVarP(
		&flags.force, "force", "f", false,
		"Force overwrite of existing [global] config files",
	)

	flags.config.register(cmd)

	return cmd
}

func pullCmdFunc(
	cmd *cobra.Command,
	url string,
	flags *pullCmdFlags,
) error {
	box, err := devbox.Open(flags.config.path, cmd.ErrOrStderr())
	if err != nil {
		return errors.WithStack(err)
	}

	pullPath, err := absolutizeIfLocal(url)
	if err != nil {
		return errors.WithStack(err)
	}

	err = box.Pull(cmd.Context(), flags.force, pullPath)
	if prompt := pullErrorPrompt(err); prompt != "" {
		prompt := &survey.Confirm{Message: prompt}
		if err = survey.AskOne(prompt, &flags.force); err != nil {
			return errors.WithStack(err)
		}
		if !flags.force {
			return nil
		}
		err = box.Pull(cmd.Context(), flags.force, pullPath)
	}
	if err != nil {
		return err
	}

	return installCmdFunc(
		cmd,
		runCmdFlags{config: configFlags{path: flags.config.path}},
	)
}

func pullErrorPrompt(err error) string {
	switch {
	case errors.Is(err, fs.ErrExist):
		return "Global profile already exists. Overwrite?"
	default:
		return ""
	}
}

func absolutizeIfLocal(path string) (string, error) {
	if _, err := os.Stat(path); err == nil {
		return filepath.Abs(path)
	}
	return path, nil
}
