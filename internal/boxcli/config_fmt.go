// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"go.jetify.com/devbox/internal/devconfig"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage your devbox.json config file",
	}
	cmd.AddCommand(configFmtCmd())
	return cmd
}

type configFmtFlags struct {
	pathFlag
}

func configFmtCmd() *cobra.Command {
	flags := &configFmtFlags{}
	cmd := &cobra.Command{
		Use:   "fmt",
		Short: "Format and modernize devbox.json",
		Long: "Format and modernize devbox.json. This rewrites the config using a " +
			"canonical layout and migrates deprecated fields (such as the nested " +
			`"shell" object) to their modern, top-level equivalents.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return configFmtFunc(cmd, flags)
		},
	}
	flags.pathFlag.register(cmd)
	return cmd
}

func configFmtFunc(cmd *cobra.Command, flags *configFmtFlags) error {
	path := flags.path
	if path == "" {
		path = "."
	}

	// Open the config directly (rather than through devbox.Open) so that
	// formatting doesn't trigger unrelated environment setup or warnings.
	cfg, err := devconfig.Open(path)
	if err != nil {
		return err
	}

	// Modernize: migrate the deprecated nested "shell" object to top-level
	// init_hook and scripts fields.
	cfg.Root.MigrateShell()

	// Save writes the (re)formatted config back to disk.
	dir := filepath.Dir(cfg.Root.AbsRootPath)
	if err := cfg.Root.SaveTo(dir); err != nil {
		return err
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Formatted %s\n", cfg.Root.AbsRootPath)
	return nil
}
