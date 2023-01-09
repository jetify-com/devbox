// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

// Package devbox creates isolated development environments.
package impl

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/boxcli/generate"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/fileutil"
	"go.jetpack.io/devbox/internal/initrec"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/planner"
	"go.jetpack.io/devbox/internal/planner/plansdk"
	"go.jetpack.io/devbox/internal/plugin"
	"golang.org/x/exp/slices"
)

const (
	// configFilename is name of the JSON file that defines a devbox environment.
	configFilename = "devbox.json"

	// shellHistoryFile keeps the history of commands invoked inside devbox shell
	shellHistoryFile = ".devbox/shell_history"
)

func InitConfig(dir string, writer io.Writer) (created bool, err error) {
	cfgPath := filepath.Join(dir, configFilename)

	config := &Config{
		Nixpkgs: NixpkgsConfig{
			Commit: plansdk.DefaultNixpkgsCommit,
		},
	}

	pkgsToSuggest, err := initrec.Get(dir)
	if err != nil {
		return false, err
	}
	if len(pkgsToSuggest) > 0 {
		s := fmt.Sprintf("devbox add %s", strings.Join(pkgsToSuggest, " "))
		fmt.Fprintf(
			writer,
			"We detected extra packages you may need. To install them, run `%s`\n",
			color.HiYellowString(s),
		)
	}
	return cuecfg.InitFile(cfgPath, config)
}

type Devbox struct {
	cfg *Config
	// configDir is the directory where the config file (devbox.json) resides
	configDir     string
	pluginManager *plugin.Manager
	writer        io.Writer
}

// TODO savil. dir is technically path since it could be a dir or file
func Open(dir string, writer io.Writer) (*Devbox, error) {

	cfgDir, err := findConfigDir(dir)
	if err != nil {
		return nil, err
	}
	cfgPath := filepath.Join(cfgDir, configFilename)

	cfg, err := ReadConfig(cfgPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if err = upgradeConfig(cfg, cfgPath); err != nil {
		return nil, err
	}

	box := &Devbox{
		cfg:           cfg,
		configDir:     cfgDir,
		pluginManager: plugin.NewManager(),
		writer:        writer,
	}
	return box, nil
}

func (d *Devbox) ConfigDir() string {
	return d.configDir
}

func (d *Devbox) Config() *Config {
	return d.cfg
}

func (d *Devbox) Add(pkgs ...string) error {
	// Check packages are valid before adding.
	for _, pkg := range pkgs {
		ok := nix.PkgExists(d.cfg.Nixpkgs.Commit, pkg)
		if !ok {
			return errors.WithMessage(nix.ErrPackageNotFound, pkg)
		}
	}

	// Add to Packages to config only if it's not already there
	for _, pkg := range pkgs {
		if slices.Contains(d.cfg.Packages, pkg) {
			continue
		}
		d.cfg.Packages = append(d.cfg.Packages, pkg)
	}
	if err := d.saveCfg(); err != nil {
		return err
	}

	d.pluginManager.ApplyOptions(plugin.WithAddMode())
	if err := d.ensurePackagesAreInstalled(install); err != nil {
		return err
	}
	if featureflag.PKGConfig.Enabled() {
		for _, pkg := range pkgs {
			if err := plugin.PrintReadme(
				pkg,
				d.configDir,
				d.writer,
				IsDevboxShellEnabled(),
				false, /*markdown*/
			); err != nil {
				return err
			}
		}
	}
	return d.printPackageUpdateMessage(install, pkgs)
}

func (d *Devbox) Remove(pkgs ...string) error {

	// First, save which packages are being uninstalled. Do this before we modify d.cfg.Packages below.
	uninstalledPackages := lo.Intersect(d.cfg.Packages, pkgs)

	var missingPkgs []string
	d.cfg.Packages, missingPkgs = lo.Difference(d.cfg.Packages, pkgs)

	if len(missingPkgs) > 0 {
		fmt.Fprintf(
			d.writer,
			"%s the following packages were not found in your devbox.json: %s\n",
			color.HiYellowString("Warning:"),
			strings.Join(missingPkgs, ", "),
		)
	}
	if err := d.saveCfg(); err != nil {
		return err
	}

	if featureflag.PKGConfig.Enabled() {
		if err := plugin.Remove(d.configDir, uninstalledPackages); err != nil {
			return err
		}
	}

	if err := d.ensurePackagesAreInstalled(uninstall); err != nil {
		return err
	}

	return d.printPackageUpdateMessage(uninstall, uninstalledPackages)
}

func (d *Devbox) ShellPlan() (*plansdk.ShellPlan, error) {
	userDefinedPkgs := d.cfg.Packages
	shellPlan := planner.GetShellPlan(d.configDir, userDefinedPkgs)
	shellPlan.DevPackages = userDefinedPkgs

	if nixpkgsInfo, err := plansdk.GetNixpkgsInfo(d.cfg.Nixpkgs.Commit); err != nil {
		return nil, err
	} else {
		shellPlan.NixpkgsInfo = nixpkgsInfo
	}

	return shellPlan, nil
}

func (d *Devbox) Generate() error {
	if err := d.generateShellFiles(); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (d *Devbox) Shell() error {
	if err := d.ensurePackagesAreInstalled(install); err != nil {
		return err
	}
	fmt.Fprintln(d.writer, "Starting a devbox shell...")

	profileDir, err := d.profileDir()
	if err != nil {
		return err
	}

	nixShellFilePath := filepath.Join(d.configDir, ".devbox/gen/shell.nix")

	pluginHooks, err := plugin.InitHooks(d.cfg.Packages, d.configDir)
	if err != nil {
		return err
	}

	opts := []nix.ShellOption{
		nix.WithPluginInitHook(strings.Join(pluginHooks, "\n")),
		nix.WithProfile(profileDir),
		nix.WithHistoryFile(filepath.Join(d.configDir, shellHistoryFile)),
		nix.WithConfigDir(d.configDir),
	}
	// TODO: separate package suggestions from shell planners
	if featureflag.PKGConfig.Enabled() {
		env, err := plugin.Env(d.cfg.Packages, d.configDir)
		if err != nil {
			return err
		}
		opts = append(
			opts,
			nix.WithEnvVariables(env),
			nix.WithPKGConfigDir(filepath.Join(d.configDir, plugin.VirtenvBinPath)),
		)
	}

	shell, err := nix.DetectShell(opts...)
	if err != nil {
		// Fall back to using a plain Nix shell.
		shell = &nix.Shell{}
	}

	shell.UserInitHook = d.cfg.Shell.InitHook.String()
	return shell.Run(nixShellFilePath)
}

func (d *Devbox) RunScriptInShell(scriptName string) error {
	profileDir, err := d.profileDir()
	if err != nil {
		return err
	}

	script := d.cfg.Shell.Scripts[scriptName]
	if script == nil {
		return errors.Errorf("unable to find a script with name %s", scriptName)
	}

	shell, err := nix.DetectShell(
		nix.WithProfile(profileDir),
		nix.WithHistoryFile(filepath.Join(d.configDir, shellHistoryFile)),
		nix.WithUserScript(scriptName, script.String()),
		nix.WithConfigDir(d.configDir),
	)

	if err != nil {
		fmt.Fprint(d.writer, err)
		shell = &nix.Shell{}
	}

	return shell.RunInShell()
}

// TODO: consider unifying the implementations of RunScript and Shell.
func (d *Devbox) RunScript(scriptName string) error {
	if err := d.ensurePackagesAreInstalled(install); err != nil {
		return err
	}
	fmt.Fprintln(d.writer, "Starting a devbox shell...")

	profileDir, err := d.profileDir()
	if err != nil {
		return err
	}

	nixShellFilePath := filepath.Join(d.configDir, ".devbox/gen/shell.nix")
	script := d.cfg.Shell.Scripts[scriptName]
	if script == nil {
		return errors.Errorf("unable to find a script with name %s", scriptName)
	}

	pluginHooks, err := plugin.InitHooks(d.cfg.Packages, d.configDir)
	if err != nil {
		return err
	}

	opts := []nix.ShellOption{
		nix.WithPluginInitHook(strings.Join(pluginHooks, "\n")),
		nix.WithProfile(profileDir),
		nix.WithHistoryFile(filepath.Join(d.configDir, shellHistoryFile)),
		nix.WithUserScript(scriptName, script.String()),
		nix.WithConfigDir(d.configDir),
	}

	if featureflag.PKGConfig.Enabled() {
		env, err := plugin.Env(d.cfg.Packages, d.configDir)
		if err != nil {
			return err
		}
		opts = append(
			opts,
			nix.WithEnvVariables(env),
			nix.WithPKGConfigDir(filepath.Join(d.configDir, plugin.VirtenvBinPath)),
		)
	}

	shell, err := nix.DetectShell(opts...)

	if err != nil {
		fmt.Fprint(d.writer, err)
		shell = &nix.Shell{}
	}

	shell.UserInitHook = d.cfg.Shell.InitHook.String()
	return shell.Run(nixShellFilePath)
}

func (d *Devbox) ListScripts() []string {
	keys := make([]string, len(d.cfg.Shell.Scripts))
	i := 0
	for k := range d.cfg.Shell.Scripts {
		keys[i] = k
		i++
	}
	return keys
}

func (d *Devbox) Exec(cmds ...string) error {
	if err := d.ensurePackagesAreInstalled(install); err != nil {
		return err
	}

	profileBinDir, err := d.profileBinDir()
	if err != nil {
		return err
	}

	env := []string{}
	virtenvBinPath := ""
	if featureflag.PKGConfig.Enabled() {
		envMap, err := plugin.Env(d.cfg.Packages, d.configDir)
		if err != nil {
			return err
		}
		for k, v := range envMap {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		virtenvBinPath = filepath.Join(d.configDir, plugin.VirtenvBinPath) + ":"
	}
	pathWithProfileBin := fmt.Sprintf("PATH=%s%s:$PATH", virtenvBinPath, profileBinDir)
	cmds = append([]string{pathWithProfileBin}, cmds...)

	nixDir := filepath.Join(d.configDir, ".devbox/gen/shell.nix")
	return nix.Exec(nixDir, cmds, env)
}

func (d *Devbox) PrintShellEnv() error {
	profileBinDir, err := d.profileBinDir()
	if err != nil {
		return errors.WithStack(err)
	}
	// TODO: For now we just updated the PATH but this may need to evolve
	// to essentially a parsed shellrc.tmpl
	fmt.Fprintf(d.writer, "export PATH=\"%s:$PATH\"", profileBinDir)
	return nil
}

func (d *Devbox) Info(pkg string, markdown bool) error {
	info, hasInfo := nix.PkgInfo(d.cfg.Nixpkgs.Commit, pkg)
	if !hasInfo {
		_, err := fmt.Fprintf(d.writer, "Package %s not found\n", pkg)
		return errors.WithStack(err)
	}
	if _, err := fmt.Fprintf(
		d.writer,
		"%s%s\n",
		lo.Ternary(markdown, "## ", ""),
		info,
	); err != nil {
		return errors.WithStack(err)
	}
	return plugin.PrintReadme(
		pkg,
		d.configDir,
		d.writer,
		false, /*showSourceEnv*/
		markdown,
	)
}

// generates devcontainer.json and Dockerfile for vscode run-in-container
// and Github Codespaces
func (d *Devbox) GenerateDevcontainer(force bool) error {
	// construct path to devcontainer directory
	devContainerPath := filepath.Join(d.configDir, ".devcontainer/")
	devContainerJSONPath := filepath.Join(devContainerPath, "devcontainer.json")
	dockerfilePath := filepath.Join(devContainerPath, "Dockerfile")

	// check if devcontainer.json or Dockerfile exist
	filesExist := plansdk.FileExists(devContainerJSONPath) || plansdk.FileExists(dockerfilePath)

	if force || !filesExist {
		// create directory
		err := os.MkdirAll(devContainerPath, os.ModePerm)
		if err != nil {
			return errors.WithStack(err)
		}
		// generate dockerfile
		err = generate.CreateDockerfile(tmplFS, devContainerPath)
		if err != nil {
			return errors.WithStack(err)
		}
		// generate devcontainer.json
		err = generate.CreateDevcontainer(devContainerPath, d.cfg.Packages)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		return usererr.New(
			"Files devcontainer.json or Dockerfile are already present in .devcontainer/. " +
				"Remove the files or use --force to overwrite them.",
		)
	}
	return nil
}

// generates a Dockerfile that replicates the devbox shell
func (d *Devbox) GenerateDockerfile(force bool) error {
	dockerfilePath := filepath.Join(d.configDir, "Dockerfile")
	// check if Dockerfile doesn't exist
	filesExist := plansdk.FileExists(dockerfilePath)
	if force || !filesExist {
		// generate dockerfile
		err := generate.CreateDockerfile(tmplFS, d.configDir)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		return usererr.New(
			"Dockerfile is already present in the current directory. " +
				"Remove it or use --force to overwrite it.",
		)
	}

	return nil
}

// generates a .envrc file that makes direnv integration convenient
func (d *Devbox) GenerateEnvrc(force bool) error {
	envrcfilePath := filepath.Join(d.configDir, ".envrc")
	filesExist := fileutil.Exists(envrcfilePath)
	// confirm .envrc doesn't exist and don't overwrite an existing .envrc
	if force || !filesExist {
		err := generate.CreateEnvrc(tmplFS, d.configDir)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		return usererr.New(
			"A .envrc is already present in the current directory. " +
				"Remove it or use --force to overwrite it.",
		)
	}

	return nil
}

// saveCfg writes the config file to the devbox directory.
func (d *Devbox) saveCfg() error {
	cfgPath := filepath.Join(d.configDir, configFilename)
	return cuecfg.WriteFile(cfgPath, d.cfg)
}

func (d *Devbox) Services() (plugin.Services, error) {
	return plugin.GetServices(d.cfg.Packages, d.configDir)
}

func (d *Devbox) StartServices(services ...string) error {
	if !IsDevboxShellEnabled() {
		return d.Exec(append([]string{"devbox", "services", "start"}, services...)...)
	}
	return plugin.StartServices(d.cfg.Packages, services, d.configDir, d.writer)
}

func (d *Devbox) StopServices(services ...string) error {
	if !IsDevboxShellEnabled() {
		return d.Exec(append([]string{"devbox", "services", "stop"}, services...)...)
	}
	return plugin.StopServices(d.cfg.Packages, services, d.configDir, d.writer)
}

func (d *Devbox) generateShellFiles() error {
	plan, err := d.ShellPlan()
	if err != nil {
		return err
	}
	return generateForShell(d.configDir, plan, d.pluginManager)
}

func (d *Devbox) profileDir() (string, error) {
	absPath := filepath.Join(d.configDir, nix.ProfilePath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return "", errors.WithStack(err)
	}

	return absPath, nil
}

func (d *Devbox) profileBinDir() (string, error) {
	profileDir, err := d.profileDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(profileDir, "bin"), nil
}

// installMode is an enum for helping with ensurePackagesAreInstalled implementation
type installMode string

const (
	install   installMode = "install"
	uninstall installMode = "uninstall"
)

func (d *Devbox) ensurePackagesAreInstalled(mode installMode) error {
	if err := d.generateShellFiles(); err != nil {
		return err
	}

	installingVerb := "Installing"
	if mode == uninstall {
		installingVerb = "Uninstalling"
	}
	_, _ = fmt.Fprintf(d.writer, "%s nix packages. This may take a while... ", installingVerb)

	// We need to re-install the packages
	if err := d.installNixProfile(); err != nil {
		fmt.Fprintln(d.writer)
		return errors.Wrap(err, "apply Nix derivation")
	}
	fmt.Fprintln(d.writer, "done.")

	if featureflag.PKGConfig.Enabled() {
		if err := plugin.RemoveInvalidSymlinks(d.configDir); err != nil {
			return err
		}
	}

	return nil
}

func (d *Devbox) printPackageUpdateMessage(
	mode installMode,
	pkgs []string,
) error {
	verb := "installed"
	var infos []*nix.Info
	for _, pkg := range pkgs {
		info, _ := nix.PkgInfo(d.cfg.Nixpkgs.Commit, pkg)
		infos = append(infos, info)
	}
	if mode == uninstall {
		verb = "removed"
	}

	if len(pkgs) > 0 {

		successMsg := fmt.Sprintf("%s (%s) is now %s.", pkgs[0], infos[0], verb)
		if len(pkgs) > 1 {
			pkgsWithVersion := []string{}
			for idx, pkg := range pkgs {
				pkgsWithVersion = append(
					pkgsWithVersion,
					fmt.Sprintf("%s (%s)", pkg, infos[idx]),
				)
			}
			successMsg = fmt.Sprintf(
				"%s are now %s.",
				strings.Join(pkgsWithVersion, ", "),
				verb,
			)
		}
		fmt.Fprint(d.writer, successMsg)

		// (Only when in devbox shell) Prompt the user to run `hash -r` to ensure
		// their shell can access the most recently installed binaries, or ensure
		// their recently uninstalled binaries are not accidentally still available.
		if !IsDevboxShellEnabled() {
			fmt.Fprintln(d.writer)
		} else {
			fmt.Fprintln(d.writer, " Run `hash -r` to ensure your shell is updated.")
		}
	} else {
		fmt.Fprintf(d.writer, "No packages %s.\n", verb)
	}
	return nil
}

// installNixProfile installs or uninstalls packages to or from this
// devbox's Nix profile so that it matches what's in development.nix or flake.nix
func (d *Devbox) installNixProfile() (err error) {
	profileDir, err := d.profileDir()
	if err != nil {
		return err
	}

	var cmd *exec.Cmd
	if featureflag.Flakes.Enabled() {
		cmd = d.installNixProfileFlakeCommand(profileDir)
		defer func() {
			if err == nil {
				_ = d.copyFlakeLockToDevboxLock()
			}
		}()
	} else {
		cmd = exec.Command(
			"nix-env",
			"--profile", profileDir,
			"--install",
			"-f", filepath.Join(d.configDir, ".devbox/gen/development.nix"),
		)
	}

	cmd.Env = nix.DefaultEnv()

	debug.Log("Running command: %s\n", cmd.Args)
	_, err = cmd.Output()

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return errors.Errorf("running command %s: exit status %d with command stderr: %s",
			cmd, exitErr.ExitCode(), string(exitErr.Stderr))
	}
	if err != nil {
		return errors.Errorf("running command %s: %v", cmd, err)
	}

	return
}

// Move to a utility package?
func IsDevboxShellEnabled() bool {
	inDevboxShell, err := strconv.ParseBool(os.Getenv("DEVBOX_SHELL_ENABLED"))
	if err != nil {
		return false
	}
	return inDevboxShell
}
