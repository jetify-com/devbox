// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

// Package devbox creates isolated development environments.
package devbox

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
	"go.jetpack.io/devbox/boxcli/featureflag"
	"go.jetpack.io/devbox/boxcli/generate"
	"go.jetpack.io/devbox/boxcli/usererr"
	"go.jetpack.io/devbox/cuecfg"
	"go.jetpack.io/devbox/debug"
	"go.jetpack.io/devbox/nix"
	"go.jetpack.io/devbox/pkgcfg"
	"go.jetpack.io/devbox/planner"
	"go.jetpack.io/devbox/planner/plansdk"
	"golang.org/x/exp/slices"
)

const (
	// configFilename is name of the JSON file that defines a devbox environment.
	configFilename = "devbox.json"

	// shellHistoryFile keeps the history of commands invoked inside devbox shell
	shellHistoryFile = ".devbox/shell_history"
)

// InitConfig creates a default devbox config file if one doesn't already
// exist.
func InitConfig(dir string) (created bool, err error) {
	cfgPath := filepath.Join(dir, configFilename)

	config := &Config{
		Nixpkgs: NixpkgsConfig{
			Commit: plansdk.DefaultNixpkgsCommit,
		},
	}
	return cuecfg.InitFile(cfgPath, config)
}

// Devbox provides an isolated development environment that contains a set of
// Nix packages.
type Devbox struct {
	cfg *Config
	// configDir is the directory where the config file (devbox.json) resides
	configDir string
	writer    io.Writer
}

// Open opens a devbox by reading the config file in dir.
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
		cfg:       cfg,
		configDir: cfgDir,
		writer:    writer,
	}
	return box, nil
}

func (d *Devbox) ConfigDir() string {
	return d.configDir
}

func (d *Devbox) Config() *Config {
	return d.cfg
}

// Add adds a Nix package to the config so that it's available in the devbox
// environment. It validates that the Nix package exists, but doesn't install
// it. Adding a duplicate package is a no-op.
func (d *Devbox) Add(pkgs ...string) error {
	// Check packages are valid before adding.
	for _, pkg := range pkgs {
		ok := nix.PkgExists(d.cfg.Nixpkgs.Commit, pkg)
		if !ok {
			return errors.Errorf("package %s not found", pkg)
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

	if err := d.ensurePackagesAreInstalled(install); err != nil {
		return err
	}
	if featureflag.PKGConfig.Enabled() {
		for _, pkg := range pkgs {
			if err := pkgcfg.PrintReadme(
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

// Remove removes Nix packages from the config so that it no longer exists in
// the devbox environment.
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
		if err := pkgcfg.Remove(d.configDir, uninstalledPackages); err != nil {
			return err
		}
	}

	if err := d.ensurePackagesAreInstalled(uninstall); err != nil {
		return err
	}

	return d.printPackageUpdateMessage(uninstall, uninstalledPackages)
}

// ShellPlan creates a plan of the actions that devbox will take to generate its
// shell environment.
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

// Generate creates the directory of Nix files and the Dockerfile that define
// the devbox environment.
func (d *Devbox) Generate() error {
	if err := d.generateShellFiles(); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// Shell generates the devbox environment and launches nix-shell as a child
// process.
func (d *Devbox) Shell() error {
	if err := d.ensurePackagesAreInstalled(install); err != nil {
		return err
	}
	plan, err := d.ShellPlan()
	if err != nil {
		return err
	}
	profileDir, err := d.profileDir()
	if err != nil {
		return err
	}

	nixShellFilePath := filepath.Join(d.configDir, ".devbox/gen/shell.nix")

	opts := []nix.ShellOption{
		nix.WithPlanInitHook(strings.Join(plan.ShellInitHook, "\n")),
		nix.WithProfile(profileDir),
		nix.WithHistoryFile(filepath.Join(d.configDir, shellHistoryFile)),
		nix.WithConfigDir(d.configDir),
	}

	if featureflag.PKGConfig.Enabled() {
		env, err := pkgcfg.Env(plan.DevPackages, d.configDir)
		if err != nil {
			return err
		}
		opts = append(
			opts,
			nix.WithEnvVariables(env),
			nix.WithPKGConfigDir(filepath.Join(d.configDir, ".devbox/conf/bin")),
		)
	}

	shell, err := nix.DetectShell(opts...)
	if err != nil {
		// Fall back to using a plain Nix shell.
		shell = &nix.Shell{}
	}

	allPkgs := planner.GetShellPackageSuggestion(d.configDir, d.cfg.Packages)
	pkgsToSuggest, _ := lo.Difference(allPkgs, d.cfg.Packages)
	if len(pkgsToSuggest) > 0 {
		s := fmt.Sprintf("devbox add %s", strings.Join(pkgsToSuggest, " "))
		fmt.Fprintf(
			d.writer,
			"We detected extra packages you may need. To install them, run `%s`\n",
			color.HiYellowString(s),
		)
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
		fmt.Print(err)
		shell = &nix.Shell{}
	}

	return shell.RunInShell()
}

// TODO: consider unifying the implementations of RunScript and Shell.
func (d *Devbox) RunScript(scriptName string) error {
	if err := d.ensurePackagesAreInstalled(install); err != nil {
		return err
	}

	plan, err := d.ShellPlan()
	if err != nil {
		return err
	}
	profileDir, err := d.profileDir()
	if err != nil {
		return err
	}

	nixShellFilePath := filepath.Join(d.configDir, ".devbox/gen/shell.nix")
	script := d.cfg.Shell.Scripts[scriptName]
	if script == nil {
		return errors.Errorf("unable to find a script with name %s", scriptName)
	}

	opts := []nix.ShellOption{
		nix.WithPlanInitHook(strings.Join(plan.ShellInitHook, "\n")),
		nix.WithProfile(profileDir),
		nix.WithHistoryFile(filepath.Join(d.configDir, shellHistoryFile)),
		nix.WithUserScript(scriptName, script.String()),
		nix.WithConfigDir(d.configDir),
	}

	if featureflag.PKGConfig.Enabled() {
		env, err := pkgcfg.Env(plan.DevPackages, d.configDir)
		if err != nil {
			return err
		}
		opts = append(
			opts,
			nix.WithEnvVariables(env),
			nix.WithPKGConfigDir(filepath.Join(d.configDir, ".devbox/conf/bin")),
		)
	}

	shell, err := nix.DetectShell(opts...)

	if err != nil {
		fmt.Print(err)
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
	confBinPath := ""
	if featureflag.PKGConfig.Enabled() {
		envMap, err := pkgcfg.Env(d.cfg.Packages, d.configDir)
		if err != nil {
			return err
		}
		for k, v := range envMap {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		confBinPath = filepath.Join(d.configDir, ".devbox/conf/bin") + ":"
	}
	pathWithProfileBin := fmt.Sprintf("PATH=%s%s:$PATH", confBinPath, profileBinDir)
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
	return pkgcfg.PrintReadme(
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
	// check if Dockerfile doesn't exits
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

// saveCfg writes the config file to the devbox directory.
func (d *Devbox) saveCfg() error {
	cfgPath := filepath.Join(d.configDir, configFilename)
	return cuecfg.WriteFile(cfgPath, d.cfg)
}

func (d *Devbox) Services() (pkgcfg.Services, error) {
	return pkgcfg.GetServices(d.cfg.Packages, d.configDir)
}

func (d *Devbox) StartService(serviceName string) error {
	if !IsDevboxShellEnabled() {
		return d.Exec("devbox", "services", "start", serviceName)
	}
	return pkgcfg.StartService(d.cfg.Packages, serviceName, d.configDir, d.writer)
}

func (d *Devbox) StopService(serviceName string) error {
	if !IsDevboxShellEnabled() {
		return d.Exec("devbox", "services", "stop", serviceName)
	}
	return pkgcfg.StopService(d.cfg.Packages, serviceName, d.configDir, d.writer)
}

func (d *Devbox) generateShellFiles() error {
	plan, err := d.ShellPlan()
	if err != nil {
		return err
	}
	return generateForShell(d.configDir, plan)
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
	if err := d.applyDevNixDerivation(); err != nil {
		fmt.Println()
		return errors.Wrap(err, "apply Nix derivation")
	}
	fmt.Println("done.")

	if featureflag.PKGConfig.Enabled() {
		if err := pkgcfg.RemoveInvalidSymlinks(d.configDir); err != nil {
			return err
		}
	}

	return nil
}

func (d *Devbox) printPackageUpdateMessage(mode installMode, pkgs []string) error {
	installedVerb := "installed"
	if mode == uninstall {
		installedVerb = "removed"
	}

	if len(pkgs) > 0 {

		successMsg := fmt.Sprintf("%s is now %s.", pkgs[0], installedVerb)
		if len(pkgs) > 1 {
			successMsg = fmt.Sprintf("%s are now %s.", strings.Join(pkgs, ", "), installedVerb)
		}
		fmt.Fprint(d.writer, successMsg)

		// (Only when in devbox shell) Prompt the user to run `hash -r` to ensure their
		// shell can access the most recently installed binaries, or ensure their
		// recently uninstalled binaries are not accidentally still available.
		if !IsDevboxShellEnabled() {
			fmt.Fprintln(d.writer)
		} else {
			fmt.Fprintln(d.writer, " Run `hash -r` to ensure your shell is updated.")
		}
	} else {
		fmt.Fprintf(d.writer, "No packages %s.\n", installedVerb)
	}
	return nil
}

// applyDevNixDerivation installs or uninstalls packages to or from this
// devbox's Nix profile so that it matches what's in development.nix.
func (d *Devbox) applyDevNixDerivation() error {
	profileDir, err := d.profileDir()
	if err != nil {
		return err
	}

	cmd := exec.Command("nix-env",
		"--profile", profileDir,
		"--install",
		"-f", filepath.Join(d.configDir, ".devbox/gen/development.nix"),
	)

	debug.Log("Running command: %s\n", cmd.Args)
	_, err = cmd.Output()

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return errors.Errorf("running command %s: exit status %d with command output: %s",
			cmd, exitErr.ExitCode(), string(exitErr.Stderr))
	}
	if err != nil {
		return errors.Errorf("running command %s: %v", cmd, err)
	}
	return nil
}

// Move to a utility package?
func IsDevboxShellEnabled() bool {
	inDevboxShell, err := strconv.ParseBool(os.Getenv("DEVBOX_SHELL_ENABLED"))
	if err != nil {
		return false
	}
	return inDevboxShell
}
