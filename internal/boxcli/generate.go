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
		&flags.printEnvrcContent, "print-envrc", "p", false, "output contents of devbox configuration to use in .envrc")
	// this command marks a flag as hidden. Error handling for it is not necessary.
	_ = command.Flags().MarkHidden("print-envrc")

	// --envrc-dir allows users to specify a directory where the .envrc file should be generated
	// separately from the devbox config directory. Without this flag, the .envrc file
	// will be generated in the same directory as the devbox config file (i.e., either the current
	// directory or the directory specified by --config). This is useful for users who want to keep
	// their .envrc and devbox config files in different locations.
	command.Flags().StringVar(
		&flags.envrcDir, "envrc-dir", "", "path to directory where the .envrc file should be generated. "+
			"If not specified, the .envrc file will be generated in the current working directory.")

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

	// Determine the directories for .envrc and config
	configDir, envrcDir, err := determineDirenvDirs(flags.config.path, flags.envrcDir)
	if err != nil {
		return errors.WithStack(err)
	}

	generateOpts := devopt.EnvrcOpts{
		EnvrcDir:  envrcDir,
		ConfigDir: configDir,
		EnvFlags:  devopt.EnvFlags(flags.envFlag),
	}

	if flags.printEnvrcContent {
		return devbox.PrintEnvrcContent(cmd.OutOrStdout(), generateOpts)
	}

	box, err := devbox.Open(&devopt.Opts{
		Dir:         filepath.Join(envrcDir, configDir),
		Environment: flags.config.environment,
		Stderr:      cmd.ErrOrStderr(),
	})
	if err != nil {
		return errors.WithStack(err)
	}

	return box.GenerateEnvrcFile(
		cmd.Context(), flags.force, generateOpts)
}

// Returns cononical paths for configDir and envrcDir. Both locations are relative to the current
// working directory when provided to this function. However, since the config file will ultimately
// be relative to the .envrc file, we need to determine the relative path from envrcDir to configDir.
func determineDirenvDirs(configDir, envrcDir string) (string, string, error) {
	// If envrcDir is not specified, we will use the configDir as the location for .envrc. This is
	// for backward compatibility (prior to the --envrc-dir flag being introduced).
	if envrcDir == "" {
		return "", configDir, nil
	}

	// If no configDir is specified, it will be assumed to be in the same directory as the .envrc file
	// which means we can just return an empty configDir.
	if configDir == "" {
		return "", envrcDir, nil
	}

	relativeConfigDir, err := filepath.Rel(envrcDir, configDir)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to determine relative path from %s to %s", envrcDir, configDir)
	}

	// If the relative path is ".", it means configDir is the same as envrcDir. Leaving it as "."
	// will result in the .envrc containing "--config .", which is fine, but unnecessary and also
	// a change from the previous behavior. So we will return an empty string for relativeConfigDir
	// which will result in the .envrc file not containing the "--config" flag at all.
	if relativeConfigDir == "." {
		relativeConfigDir = ""
	}

	return relativeConfigDir, envrcDir, nil
}
