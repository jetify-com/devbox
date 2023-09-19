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
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/goutil"
	"go.jetpack.io/devbox/internal/impl/devopt"
	"go.jetpack.io/devbox/internal/pullbox/s3"
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
		Args:    cobra.MaximumNArgs(1),
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			return pullCmdFunc(cmd, goutil.GetDefaulted(args, 0), &flags)
		},
	}

	cmd.Flags().BoolVarP(
		&flags.force, "force", "f", false,
		"Force overwrite of existing [global] config files",
	)

	flags.config.register(cmd)

	return cmd
}

func pullCmdFunc(cmd *cobra.Command, url string, flags *pullCmdFlags) error {
	box, err := devbox.Open(&devopt.Opts{
		Dir:    flags.config.path,
		Writer: cmd.ErrOrStderr(),
	})
	if err != nil {
		return errors.WithStack(err)
	}

	pullPath, err := absolutizeIfLocal(url)
	if err != nil {
		return errors.WithStack(err)
	}

	var creds devopt.Credentials
	t, err := genSession()
	if err != nil {
		return errors.WithStack(err)
	} else if t != nil {
		creds = devopt.Credentials{
			IDToken: t.IDToken,
			Email:   t.IDClaims().Email,
			Sub:     t.IDClaims().ID,
		}
	}

	err = box.Pull(cmd.Context(), devopt.PullboxOpts{
		URL:         pullPath,
		Overwrite:   flags.force,
		Credentials: creds,
	})
	if prompt := pullErrorPrompt(err); prompt != "" {
		prompt := &survey.Confirm{Message: prompt}
		if err = survey.AskOne(prompt, &flags.force); err != nil {
			return errors.WithStack(err)
		}
		if !flags.force {
			return nil
		}
		err = box.Pull(cmd.Context(), devopt.PullboxOpts{
			URL:         pullPath,
			Overwrite:   flags.force,
			Credentials: creds,
		})
	}
	if errors.Is(err, s3.ErrProfileNotFound) {
		return usererr.New(
			"Profile not found. Use `devbox global push` to create a new profile.",
		)
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
