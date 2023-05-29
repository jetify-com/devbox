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
	"go.jetpack.io/devbox/internal/pullbox/git"
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
			return pullCmdFunc(cmd, args, flags.force)
		},
		Args: cobra.ExactArgs(1),
	}

	cmd.Flags().BoolVarP(
		&flags.force, "force", "f", false,
		"Force overwrite of existing global config files",
	)

	flags.config.register(cmd)

	return cmd
}

func pullCmdFunc(
	cmd *cobra.Command,
	args []string,
	overwrite bool,
) error {
	path, err := ensureGlobalConfig(cmd)
	if err != nil {
		return errors.WithStack(err)
	}

	box, err := devbox.Open(path, cmd.ErrOrStderr())
	if err != nil {
		return errors.WithStack(err)
	}

	pullPath, err := absolutizeIfLocal(args[0])
	if err != nil {
		return errors.WithStack(err)
	}

	err = box.PullGlobal(cmd.Context(), overwrite, pullPath)
	if prompt := pullErrorPrompt(err); prompt != "" {
		prompt := &survey.Confirm{Message: prompt}
		if err = survey.AskOne(prompt, &overwrite); err != nil {
			return errors.WithStack(err)
		}
		if !overwrite {
			return nil
		}
		err = box.PullGlobal(cmd.Context(), overwrite, pullPath)
	}
	if err != nil {
		return err
	}

	return installCmdFunc(cmd, runCmdFlags{config: configFlags{path: path}})
}

func pullErrorPrompt(err error) string {
	switch {
	case errors.Is(err, fs.ErrExist):
		return "File(s) already exists. Overwrite?"
	case errors.Is(err, git.ErrExist):
		return "Directory is not empty. Overwrite?"
	case errors.Is(err, git.ErrUncommittedChanges):
		return "Uncommitted changes. Overwrite?"
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
