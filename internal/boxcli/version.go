// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"go.jetify.com/devbox/internal/build"
	"go.jetify.com/devbox/internal/envir"
	"go.jetify.com/devbox/internal/vercheck"
)

type versionFlags struct {
	verbose bool
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
