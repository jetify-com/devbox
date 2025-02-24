package boxcli

import (
	"github.com/spf13/cobra"
	"go.jetify.com/devbox/internal/patchpkg"
)

func patchCmd() *cobra.Command {
	builder := &patchpkg.DerivationBuilder{}
	cmd := &cobra.Command{
		Use:    "patch <store-path>",
		Short:  "Apply Devbox patches to a package to fix common linker errors",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return builder.Build(cmd.Context(), args[0])
		},
	}
	cmd.Flags().StringVar(&builder.Glibc, "glibc", "", "patch binaries to use a different glibc")
	cmd.Flags().StringVar(&builder.Gcc, "gcc", "", "patch binaries to use a different gcc")
	cmd.Flags().BoolVar(&builder.RestoreRefs, "restore-refs", false, "restore references to removed store paths")
	return cmd
}
