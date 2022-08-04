package docker

import (
	"fmt"
	"strings"
)

func ToArgs(args []string, opts BuildOpts) []string {
	if opts.Name != "" {
		args = append(args, "-t", opts.Name)

		for _, tag := range opts.Tags {
			args = append(args, "-t", fmt.Sprintf("%s:%s", opts.Name, tag))
		}
	}

	if len(opts.Platforms) > 0 {
		args = append(args, fmt.Sprintf("--platform=%s", strings.Join(opts.Platforms, ",")))
	}

	return args
}
