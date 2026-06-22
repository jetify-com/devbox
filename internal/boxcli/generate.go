// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"cmp"
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	"go.jetify.com/devbox/internal/boxcli/usererr"
	"go.jetify.com/devbox/internal/devbox"
	"go.jetify.com/devbox/internal/devbox/devopt"
	"go.jetify.com/devbox/internal/devbox/docgen"
	"go.jetify.com/devbox/internal/devbox/flakegen"
)

type generateCmdFlags struct {
	envFlag           // only used by generate direnv command
	config            configFlags
	force             bool
	printEnvrcContent bool
	rootUser          bool
	envrcDir          string // only used by generate direnv command
}

type generateDockerfileCmdFlags struct {
	generateCmdFlags
	forType string
}

type GenerateReadmeCmdFlags struct {
	generateCmdFlags
	saveTemplate bool
	template     string
}

type GenerateAliasCmdFlags struct {
	config   configFlags
	prefix   string
	noPrefix bool
}

type genFlakeWrapperCmdFlags struct {
	config  configFlags
	force   bool
	nixpkgs string
	attr    string
	print   bool
}

func generateCmd() *cobra.Command {
	flags := &generateCmdFlags{}

	command := &cobra.Command{
		Use:               "generate",
		Aliases:           []string{"gen"},
		Short:             "Generate supporting files for your project",
		Args:              cobra.MaximumNArgs(0),
		PersistentPreRunE: ensureNixInstalled,
	}
	command.AddCommand(genAliasCmd())
	command.AddCommand(devcontainerCmd())
	command.AddCommand(dockerfileCmd())
	command.AddCommand(debugCmd())
	command.AddCommand(direnvCmd())
	command.AddCommand(genFlakeWrapperCmd())
	command.AddCommand(genReadmeCmd())
	flags.config.register(command)

	return command
}

func debugCmd() *cobra.Command {
	flags := &generateCmdFlags{}
	command := &cobra.Command{
		Use:    "debug",
		Hidden: true,
		Args:   cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerateCmd(cmd, flags)
		},
	}
	return command
}

func devcontainerCmd() *cobra.Command {
	flags := &generateCmdFlags{}
	command := &cobra.Command{
		Use:   "devcontainer",
		Short: "Generate Dockerfile and devcontainer.json files under .devcontainer/ directory",
		Long:  "Generate Dockerfile and devcontainer.json files necessary to run VSCode in remote container environments.",
		Args:  cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerateCmd(cmd, flags)
		},
	}
	command.Flags().BoolVarP(
		&flags.force, "force", "f", false, "force overwrite on existing files")
	command.Flags().BoolVar(
		&flags.rootUser, "root-user", false, "Use root as default user inside the container")
	return command
}

func dockerfileCmd() *cobra.Command {
	flags := &generateDockerfileCmdFlags{}
	command := &cobra.Command{
		Use:   "dockerfile",
		Short: "Generate a Dockerfile that replicates devbox shell",
		Long: "Generate a Dockerfile that replicates devbox shell. " +
			"Can be used to run devbox shell environment in an OCI container.",
		Args: cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			box, err := devbox.Open(&devopt.Opts{
				Dir:         flags.config.path,
				Environment: flags.config.environment,
				Stderr:      cmd.ErrOrStderr(),
			})
			if err != nil {
				return errors.WithStack(err)
			}
			return box.GenerateDockerfile(cmd.Context(), devopt.GenerateOpts{
				ForType:  flags.forType,
				Force:    flags.force,
				RootUser: flags.rootUser,
			})
		},
	}
	command.Flags().StringVar(
		&flags.forType, "for", "dev",
		"Generate Dockerfile for a specific type of container (dev, prod)")
	command.Flag("for").Hidden = true
	command.Flags().BoolVarP(
		&flags.force, "force", "f", false, "force overwrite existing files")
	command.Flags().BoolVar(
		&flags.rootUser, "root-user", false, "Use root as default user inside the container")
	flags.config.register(command)
	return command
}

func direnvCmd() *cobra.Command {
	flags := &generateCmdFlags{}
	command := &cobra.Command{
		Use:   "direnv",
		Short: "Generate a .envrc file that integrates direnv with this devbox project",
		Long: "Generate a .envrc file that integrates direnv with this devbox project. " +
			"Requires direnv to be installed.",
		Args: cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerateDirenvCmd(cmd, flags)
		},
	}
	flags.envFlag.register(command)
	command.Flags().BoolVarP(
		&flags.force, "force", "f", false, "force overwrite existing files")
	command.Flags().BoolVarP(
		&flags.printEnvrcContent, "print-envrc", "p", false,
		"output contents of devbox configuration to use in .envrc")
	// this command marks a flag as hidden. Error handling for it is not necessary.
	_ = command.Flags().MarkHidden("print-envrc")

	// --envrc-dir allows users to specify a directory where the .envrc file should be generated
	// separately from the devbox config directory. Without this flag, the .envrc file
	// will be generated in the same directory as the devbox config file (i.e., either the current
	// directory or the directory specified by --config). This flag is useful for users who want to
	// keep their .envrc and devbox config files in different locations.
	command.Flags().StringVar(
		&flags.envrcDir, "envrc-dir", "",
		"path to directory where the .envrc file should be generated.\n"+
			"If not specified, the .envrc file will be generated in the same directory as\n"+
			"the devbox.json.")

	flags.config.register(command)
	return command
}

func genReadmeCmd() *cobra.Command {
	flags := &GenerateReadmeCmdFlags{}

	command := &cobra.Command{
		Use:   "readme [filename]",
		Short: "Generate markdown readme file for this project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			box, err := devbox.Open(&devopt.Opts{
				Dir:         flags.config.path,
				Environment: flags.config.environment,
				Stderr:      cmd.ErrOrStderr(),
			})
			if err != nil {
				return errors.WithStack(err)
			}
			outPath := ""
			if len(args) > 0 {
				outPath = args[0]
			}
			if flags.saveTemplate {
				return docgen.SaveDefaultReadmeTemplate(outPath)
			}
			return docgen.GenerateReadme(box, outPath, flags.template)
		},
	}
	flags.config.register(command)
	command.Flags().BoolVar(
		&flags.saveTemplate, "save-template", false, "Save default template for the README file")
	command.Flags().StringVarP(
		&flags.template, "template", "t", "", "Path to a custom template for the README file")

	return command
}

func genAliasCmd() *cobra.Command {
	flags := &GenerateAliasCmdFlags{}

	command := &cobra.Command{
		Use:   "alias",
		Short: "Generate shell script aliases for this project",
		Long: "Generate shell script aliases for this project. " +
			"Usage is typically `eval \"$(devbox gen alias)\"`.",
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.prefix != "" && flags.noPrefix {
				return usererr.New(
					"Cannot use both --prefix and --no-prefix flags together")
			}
			box, err := devbox.Open(&devopt.Opts{
				Dir:    flags.config.path,
				Stderr: cmd.ErrOrStderr(),
			})
			if err != nil {
				return errors.WithStack(err)
			}
			re := regexp.MustCompile("[^a-zA-Z0-9_-]+")
			prefix := cmp.Or(flags.prefix, box.Config().Root.Name)
			if prefix == "" && !flags.noPrefix {
				return usererr.New(
					"To generate aliases, you must specify a prefix, set a name " +
						"in devbox.json, or use the --no-prefix flag.")
			}
			prefix = re.ReplaceAllString(prefix, "-")
			for _, script := range box.ListScripts() {
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"alias %s%s='devbox -c \"%s\" run %s'\n",
					lo.Ternary(flags.noPrefix, "", prefix+"-"),
					script,
					box.ProjectDir(),
					script,
				)
			}
			return nil
		},
	}
	flags.config.register(command)
	command.Flags().StringVarP(
		&flags.prefix, "prefix", "p", "", "Prefix for the generated aliases")
	command.Flags().BoolVar(
		&flags.noPrefix, "no-prefix", false,
		"Do not use a prefix for the generated aliases")

	return command
}

func genFlakeWrapperCmd() *cobra.Command {
	flags := &genFlakeWrapperCmdFlags{}
	command := &cobra.Command{
		Use:   "flake-wrapper [path]",
		Short: "Generate a flake.nix wrapping an existing .nix expression",
		Long: "Generate a flake.nix next to an existing .nix expression so " +
			"the directory can be consumed as a local flake in devbox.json " +
			"(e.g. \"packages\": { \"./my-pkg\": \"\" }). The path may be a " +
			"directory containing a default.nix, or a specific .nix file. " +
			"The generated flake imports the sibling .nix file via " +
			"pkgs.callPackage.",
		Args: cobra.MaximumNArgs(1),
		// This command is pure text templating and does not need Nix.
		// Override the parent generate command's ensureNixInstalled check.
		PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "."
			if len(args) == 1 {
				target = args[0]
			}
			return runGenFlakeWrapperCmd(cmd, target, flags)
		},
	}
	flags.config.register(command)
	command.Flags().BoolVarP(
		&flags.force, "force", "f", false,
		"overwrite flake.nix if it already exists")
	command.Flags().StringVar(
		&flags.nixpkgs, "nixpkgs", "",
		"nixpkgs input URL to pin (defaults to the project's stdenv if run "+
			"inside a devbox project, else "+flakegen.DefaultNixpkgsURL+")")
	command.Flags().StringVar(
		&flags.attr, "attr", "default",
		"attribute name to expose under packages.${system}")
	command.Flags().BoolVar(
		&flags.print, "print", false,
		"print the generated flake.nix to stdout instead of writing it")
	return command
}

func runGenerateCmd(cmd *cobra.Command, flags *generateCmdFlags) error {
	// Check the directory exists.
	box, err := devbox.Open(&devopt.Opts{
		Dir:         flags.config.path,
		Environment: flags.config.environment,
		Stderr:      cmd.ErrOrStderr(),
	})
	if err != nil {
		return errors.WithStack(err)
	}
	generateOpts := devopt.GenerateOpts{
		Force:    flags.force,
		RootUser: flags.rootUser,
	}
	switch cmd.Use {
	case "debug":
		return box.Generate(cmd.Context())
	case "devcontainer":
		return box.GenerateDevcontainer(cmd.Context(), generateOpts)
	}
	return nil
}

func runGenerateDirenvCmd(cmd *cobra.Command, flags *generateCmdFlags) error {
	// --print-envrc is used within the .envrc file and therefore doesn't make sense to also
	// use it with --envrc-dir, which specifies a directory where the .envrc file should be generated.
	if flags.printEnvrcContent && flags.envrcDir != "" {
		return usererr.New(
			"Cannot use --print-envrc with --envrc-dir. " +
				"Use --envrc-dir to specify the directory where the .envrc file should be generated.")
	}

	if flags.printEnvrcContent {
		return devbox.PrintEnvrcContent(
			cmd.OutOrStdout(), devopt.EnvFlags(flags.envFlag), flags.config.path)
	}

	box, err := devbox.Open(&devopt.Opts{
		Dir:         flags.config.path,
		Environment: flags.config.environment,
		Stderr:      cmd.ErrOrStderr(),
	})
	if err != nil {
		return errors.WithStack(err)
	}

	generateEnvrcOpts := devopt.EnvrcOpts{
		EnvFlags:  devopt.EnvFlags(flags.envFlag),
		Force:     flags.force,
		EnvrcDir:  flags.envrcDir,
		ConfigDir: flags.config.path,
	}

	return box.GenerateEnvrcFile(cmd.Context(), generateEnvrcOpts)
}

func runGenFlakeWrapperCmd(
	cmd *cobra.Command,
	target string,
	flags *genFlakeWrapperCmdFlags,
) error {
	nixPath, err := flakegen.ResolveNixFile(target)
	if err != nil {
		return err
	}
	flakePath, err := flakegen.Generate(flakegen.Opts{
		NixFile:    nixPath,
		NixpkgsURL: resolveFlakeWrapperNixpkgs(cmd, flags),
		Attr:       flags.attr,
		Force:      flags.force,
		Print:      flags.print,
		Out:        cmd.OutOrStdout(),
	})
	if err != nil {
		return err
	}
	if flags.print {
		return nil
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Wrote %s.\n", flakePath)
	fmt.Fprintln(out, "Add it to devbox.json:")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "  \"packages\": {")
	fmt.Fprintf(out, "    \"./%s\": \"\"\n", filepath.Base(filepath.Dir(nixPath)))
	fmt.Fprintln(out, "  }")
	return nil
}

// resolveFlakeWrapperNixpkgs determines which nixpkgs URL to pin in the
// generated flake. An explicit --nixpkgs flag wins; otherwise, if the command
// is run inside a devbox project we use that project's stdenv so the wrapper
// matches it; otherwise fall back to flakegen.DefaultNixpkgsURL.
func resolveFlakeWrapperNixpkgs(
	cmd *cobra.Command,
	flags *genFlakeWrapperCmdFlags,
) string {
	if flags.nixpkgs != "" {
		return flags.nixpkgs
	}
	box, err := devbox.Open(&devopt.Opts{
		Dir:    flags.config.path,
		Stderr: cmd.ErrOrStderr(),
	})
	if err != nil {
		return flakegen.DefaultNixpkgsURL
	}
	stdenv := box.Stdenv().String()
	if stdenv == "" {
		return flakegen.DefaultNixpkgsURL
	}
	return stdenv
}
