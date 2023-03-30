// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/devbox/internal/vercheck"
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
	v := getVersionInfo()
	if flags.verbose {
		fmt.Fprintf(w, "Version:     %v\n", v.Version)
		fmt.Fprintf(w, "Platform:    %v\n", v.Platform)
		fmt.Fprintf(w, "Commit:      %v\n", v.Commit)
		fmt.Fprintf(w, "Commit Time: %v\n", v.CommitDate)
		fmt.Fprintf(w, "Go Version:  %v\n", v.GoVersion)
	} else {
		fmt.Fprintf(w, "%v\n", v.Version)
	}
	return nil
}

type versionInfo struct {
	Version      string
	IsPrerelease bool
	Platform     string
	Commit       string
	CommitDate   string
	GoVersion    string
}

func getVersionInfo() *versionInfo {
	v := &versionInfo{
		Version:    build.Version,
		Platform:   fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH),
		Commit:     build.Commit,
		CommitDate: build.CommitDate,
		GoVersion:  runtime.Version(),
	}

	return v
}
