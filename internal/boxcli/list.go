// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"go.jetpack.io/devbox/internal/devbox"
	"go.jetpack.io/devbox/internal/devbox/devopt"
)

type listCmdFlags struct {
	config configFlags
}

func listCmd() *cobra.Command {
	flags := listCmdFlags{}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List installed packages",
		PreRunE: ensureNixInstalled,
		RunE: func(cmd *cobra.Command, args []string) error {
			box, err := devbox.Open(&devopt.Opts{
				Dir:    flags.config.path,
				Stderr: cmd.ErrOrStderr(),
			})
			if err != nil {
				return errors.WithStack(err)
			}

			for _, pkg := range box.AllPackagesIncludingRemovedTriggerPackages() {
				resolvedVersion, err := pkg.ResolvedVersion()
				if err != nil {
					// Continue to print the package even if we can't resolve the version
					// so that the user can see the error for this package, as well as get the
					// results for the other packages
					resolvedVersion = "<error resolving version>"
				}
				msg := ""

				// Print the resolved version, unless the user has specified a version already
				if strings.HasSuffix(pkg.Versioned(), "latest") && resolvedVersion != "" {
					// Runx packages have a "v" prefix (why?). Trim for consistency.
					resolvedVersion = strings.TrimPrefix(resolvedVersion, "v")
					msg = fmt.Sprintf("* %s - %s\n", pkg.Versioned(), resolvedVersion)
				} else {
					msg = fmt.Sprintf("* %s\n", pkg.Versioned())
				}
				fmt.Fprintf(cmd.OutOrStdout(), msg)

			}
			return nil
		},
	}
	flags.config.register(cmd)
	return cmd
}
