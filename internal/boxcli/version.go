// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/build"
)

type versionFlags struct {
	verbose bool
}

func VersionCmd() *cobra.Command {
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
		"Verbose: displays additional version information",
	)
	return command
}

func versionCmdFunc(_ *cobra.Command, _ []string, flags versionFlags) error {
	v := getVersionInfo()
	if flags.verbose {
		fmt.Printf("Version:     %v\n", v.Version)
		fmt.Printf("Platform:    %v\n", v.Platform)
		fmt.Printf("Commit:      %v\n", v.Commit)
		fmt.Printf("Commit Time: %v\n", v.CommitDate)
		fmt.Printf("Go Version:  %v\n", v.GoVersion)
	} else {
		fmt.Printf("%v\n", v.Version)
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
