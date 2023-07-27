// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/wrapnix"

	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/devbox/internal/envir"
	"go.jetpack.io/devbox/internal/vercheck"
)

type versionFlags struct {
	verbose             bool
	updateDevboxSymlink bool
}

func versionCmd() *cobra.Command {
	flags := versionFlags{}
	command := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return versionCmdFunc(cmd, args, flags)
		},
	}

	command.Flags().BoolVarP(&flags.verbose, "verbose", "v", false, // value
		"displays additional version information",
	)
	// Make this flag hidden because:
	// This functionality doesn't strictly belong in this command, but we add it here
	// since `devbox version update` calls `devbox version -v` to trigger an update.
	command.Flags().BoolVarP(&flags.updateDevboxSymlink, "update-devbox-symlink", "u", false, // value
		"update the devbox symlink to point to the current binary",
	)
	_ = command.Flags().MarkHidden("update-devbox-symlink")

	command.AddCommand(selfUpdateCmd())
	return command
}

func selfUpdateCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "update",
		Short: "Update devbox launcher and binary",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return vercheck.SelfUpdate(cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	return command
}

func versionCmdFunc(cmd *cobra.Command, _ []string, flags versionFlags) error {
	w := cmd.OutOrStdout()
	info := getVersionInfo()
	if flags.verbose {
		fmt.Fprintf(w, "Version:     %v\n", info.Version)
		fmt.Fprintf(w, "Platform:    %v\n", info.Platform)
		fmt.Fprintf(w, "Commit:      %v\n", info.Commit)
		fmt.Fprintf(w, "Commit Time: %v\n", info.CommitDate)
		fmt.Fprintf(w, "Go Version:  %v\n", info.GoVersion)
		fmt.Fprintf(w, "Launcher:    %v\n", info.LauncherVersion)

		// TODO: in a subsequent PR, we should do this when flags.updateDevboxSymlink is true.
		// Not doing for now, since users who have Devbox binary prior to this edit
		// (before Devbox v0.5.9) will not invoke this flag in `devbox version update`.
		// But we still want this to run for them.
		if _, err := wrapnix.CreateDevboxSymlink(); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(w, "%v\n", info.Version)
	}
	return nil
}

type versionInfo struct {
	Version         string
	IsPrerelease    bool
	Platform        string
	Commit          string
	CommitDate      string
	GoVersion       string
	LauncherVersion string
}

func getVersionInfo() *versionInfo {
	v := &versionInfo{
		Version:         build.Version,
		Platform:        fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH),
		Commit:          build.Commit,
		CommitDate:      build.CommitDate,
		GoVersion:       runtime.Version(),
		LauncherVersion: os.Getenv(envir.LauncherVersion),
	}

	return v
}
