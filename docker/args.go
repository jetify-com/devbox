// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package docker

import (
	"fmt"
	"strings"
)

func ToArgs(args []string, flags *BuildFlags) []string {
	if flags == nil {
		return args
	}
	if args == nil {
		args = []string{}
	}
	if flags.Name != "" {
		args = append(args, "-t", flags.Name)

		for _, tag := range flags.Tags {
			args = append(args, "-t", fmt.Sprintf("%s:%s", flags.Name, tag))
		}
	}
	if flags.DockerfilePath != "" {
		args = append(args, "-f", flags.DockerfilePath)
	}
	if len(flags.Platforms) > 0 {
		args = append(args, fmt.Sprintf("--platform=%s", strings.Join(flags.Platforms, ",")))
	}

	if flags.NoCache {
		args = append(args, "--no-cache")
	}

	return args
}
