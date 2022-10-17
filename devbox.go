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
	"go.jetpack.io/devbox/cuecfg"
	"go.jetpack.io/devbox/debug"
	"go.jetpack.io/devbox/docker"
	"go.jetpack.io/devbox/nix"
	"go.jetpack.io/devbox/planner"
	"go.jetpack.io/devbox/planner/plansdk"
	"golang.org/x/exp/slices"
)

const (
	// configFilename is name of the JSON file that defines a devbox environment.
	configFilename = "devbox.json"

	// profileDir contains the contents of the profile generated via `nix-env --profile profileDir <command>`
	// Instead of using directory, prefer using the devbox.profileDir() function that ensures the directory exists.
	// TODO savil. Rename to profilePath. This is the symlink of the profile, and not a directory.
	profileDir = ".devbox/nix/profile/default"

	// shellHistoryFile keeps the history of commands invoked inside devbox shell
	shellHistoryFile = ".devbox/shell_history"
)

// InitConfig creates a default devbox config file if one doesn't already
// exist.
func InitConfig(dir string) (created bool, err error) {
	cfgPath := filepath.Join(dir, configFilename)
	return cuecfg.InitFile(cfgPath, &Config{})
}

// Devbox provides an isolated development environment that contains a set of
// Nix packages.
type Devbox struct {
	cfg *Config
	// srcDir is the directory where the config file (devbox.json) resides
	srcDir string
	writer io.Writer
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

	box := &Devbox{
		cfg:    cfg,
		srcDir: cfgDir,
		writer: writer,
	}
	return box, nil
}

// Add adds a Nix package to the config so that it's available in the devbox
// environment. It validates that the Nix package exists, but doesn't install
// it. Adding a duplicate package is a no-op.
func (d *Devbox) Add(pkgs ...string) error {
	// Check packages are valid before adding.
	for _, pkg := range pkgs {
		ok := nix.PkgExists(pkg)
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

	if err := d.ensurePackagesAreInstalled(uninstall); err != nil {
		return err
	}

	return d.printPackageUpdateMessage(uninstall, uninstalledPackages)
}

// Build creates a Docker image containing a shell with the devbox environment.
func (d *Devbox) Build(flags *docker.BuildFlags) error {
	defaultFlags := &docker.BuildFlags{
		Name:           flags.Name,
		DockerfilePath: filepath.Join(d.srcDir, ".devbox/gen", "Dockerfile"),
	}
	opts := append([]docker.BuildOptions{docker.WithFlags(defaultFlags)}, docker.WithFlags(flags))

	err := d.generateBuildFiles()
	if err != nil {
		return errors.WithStack(err)
	}
	return docker.Build(d.srcDir, opts...)
}

// Plan creates a plan of the actions that devbox will take to generate its
// shell environment.
func (d *Devbox) ShellPlan() *plansdk.ShellPlan {
	userDefinedPkgs := d.cfg.Packages
	shellPlan := planner.GetShellPlan(d.srcDir, userDefinedPkgs)
	shellPlan.DevPackages = userDefinedPkgs

	return shellPlan
}

// Plan creates a plan of the actions that devbox will take to generate its
// shell environment.
func (d *Devbox) BuildPlan() (*plansdk.BuildPlan, error) {
	userPlan := d.convertToBuildPlan()
	buildPlan, err := planner.GetBuildPlan(d.srcDir, d.cfg.Packages)
	if err != nil {
		return nil, err
	}
	return plansdk.MergeUserBuildPlan(userPlan, buildPlan)
}

// Generate creates the directory of Nix files and the Dockerfile that define
// the devbox environment.
func (d *Devbox) Generate() error {
	if err := d.generateShellFiles(); err != nil {
		return errors.WithStack(err)
	}
	if err := d.generateBuildFiles(); err != nil {
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
	plan := d.ShellPlan()
	profileDir, err := d.profileDir()
	if err != nil {
		return err
	}

	nixShellFilePath := filepath.Join(d.srcDir, ".devbox/gen/shell.nix")
	shell, err := nix.DetectShell(
		nix.WithPlanInitHook(strings.Join(plan.ShellInitHook, "\n")),
		nix.WithProfile(profileDir),
		nix.WithHistoryFile(filepath.Join(d.srcDir, shellHistoryFile)),
	)
	if err != nil {
		// Fall back to using a plain Nix shell.
		shell = &nix.Shell{}
	}

	allPkgs := planner.GetShellPackageSuggestion(d.srcDir, d.cfg.Packages)
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

func (d *Devbox) Exec(cmds ...string) error {
	if err := d.ensurePackagesAreInstalled(install); err != nil {
		return err
	}

	profileBinDir, err := d.profileBinDir()
	if err != nil {
		return err
	}

	pathWithProfileBin := fmt.Sprintf("PATH=%s:$PATH", profileBinDir)
	cmds = append([]string{pathWithProfileBin}, cmds...)

	nixDir := filepath.Join(d.srcDir, ".devbox/gen/shell.nix")
	return nix.Exec(nixDir, cmds)
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

// saveCfg writes the config file to the devbox directory.
func (d *Devbox) saveCfg() error {
	cfgPath := filepath.Join(d.srcDir, configFilename)
	return cuecfg.WriteFile(cfgPath, d.cfg)
}

func (d *Devbox) convertToBuildPlan() *plansdk.BuildPlan {
	configStages := []*Stage{d.cfg.InstallStage, d.cfg.BuildStage, d.cfg.StartStage}
	planStages := []*plansdk.Stage{{}, {}, {}}

	for i, stage := range configStages {
		if stage != nil {
			planStages[i] = &plansdk.Stage{
				Command: stage.Command,
			}
		}
	}
	return &plansdk.BuildPlan{
		DevPackages:     d.cfg.Packages,
		RuntimePackages: d.cfg.Packages,
		InstallStage:    planStages[0],
		BuildStage:      planStages[1],
		StartStage:      planStages[2],
	}
}

func (d *Devbox) generateShellFiles() error {
	return generateForShell(d.srcDir, d.ShellPlan())
}

func (d *Devbox) generateBuildFiles() error {
	// BuildPlan() will return error if plan is invalid.
	buildPlan, err := d.BuildPlan()
	if err != nil {
		return errors.WithStack(err)
	}
	if buildPlan.Warning() != nil {
		fmt.Printf("[WARNING]: %s\n", buildPlan.Warning().Error())
	}
	return generateForBuild(d.srcDir, buildPlan)
}

func (d *Devbox) profileDir() (string, error) {
	absPath := filepath.Join(d.srcDir, profileDir)
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
	fmt.Fprintf(d.writer, "%s nix packages. This may take a while...", installingVerb)

	// We need to re-install the packages
	if err := d.applyDevNixDerivation(); err != nil {
		fmt.Println()
		return errors.Wrap(err, "apply Nix derivation")
	}
	fmt.Println("done.")

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
		"-f", filepath.Join(d.srcDir, ".devbox/gen/development.nix"),
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
