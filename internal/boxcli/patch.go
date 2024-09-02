package boxcli

import (
	"github.com/spf13/cobra"
	"go.jetpack.io/devbox/internal/patchpkg"
)

func patchCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "patch <store-path>",
		Short:  "Apply Devbox patches to a package to fix common linker errors",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			builder, err := patchpkg.NewDerivationBuilder()
			if err != nil {
				return err
			}
			return builder.Build(cmd.Context(), args[0])
		},
	}
}
