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

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/boxcli/usererr"
	"go.jetpack.io/devbox/cuecfg"
	"go.jetpack.io/devbox/debug"
	"go.jetpack.io/devbox/docker"
	"go.jetpack.io/devbox/nix"
	"go.jetpack.io/devbox/pkgslice"
	"go.jetpack.io/devbox/planner"
	"go.jetpack.io/devbox/planner/plansdk"
	"golang.org/x/exp/slices"
)

const (
	// configFilename is name of the JSON file that defines a devbox environment.
	configFilename = "devbox.json"

	// profileDir contains the contents of the profile generated via `nix-env --profile profileDir <command>`
	profileDir = ".devbox/profile"

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
	// Remove packages from config.
	d.cfg.Packages = pkgslice.Exclude(d.cfg.Packages, pkgs)
	if err := d.saveCfg(); err != nil {
		return err
	}

	if err := d.ensurePackagesAreInstalled(uninstall); err != nil {
		return err
	}
	return d.printPackageUpdateMessage(uninstall, pkgs)
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
func (d *Devbox) ShellPlan() (*plansdk.Plan, error) {
	userPlan := d.convertToPlan()
	shellPlan, err := planner.GetShellPlan(d.srcDir)
	if err != nil {
		return nil, err
	}
	return plansdk.MergeUserPlan(userPlan, shellPlan)
}

// Plan creates a plan of the actions that devbox will take to generate its
// shell environment.
func (d *Devbox) BuildPlan() (*plansdk.Plan, error) {
	userPlan := d.convertToPlan()
	buildPlan, err := planner.GetBuildPlan(d.srcDir)
	if err != nil {
		return nil, err
	}
	return plansdk.MergeUserPlan(userPlan, buildPlan)
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

	plan, err := d.ShellPlan()
	if err != nil {
		return errors.WithStack(err)
	}
	nixShellFilePath := filepath.Join(d.srcDir, ".devbox/gen/shell.nix")
	sh, err := nix.DetectShell(
		nix.WithPlanInitHook(plan.ShellInitHook),
		nix.WithProfile(d.profileDir()),
		nix.WithHistoryFile(filepath.Join(d.srcDir, shellHistoryFile)),
	)
	if err != nil {
		// Fall back to using a plain Nix shell.
		sh = &nix.Shell{}
	}
	sh.UserInitHook = d.cfg.Shell.InitHook.String()
	return sh.Run(nixShellFilePath)
}

func (d *Devbox) Exec(cmds ...string) error {
	if err := d.ensurePackagesAreInstalled(install); err != nil {
		return err
	}

	pathWithProfileBin := fmt.Sprintf("PATH=%s:$PATH", d.profileBinDir())
	cmds = append([]string{pathWithProfileBin}, cmds...)

	nixDir := filepath.Join(d.srcDir, ".devbox/gen/shell.nix")
	return nix.Exec(nixDir, cmds)
}

// saveCfg writes the config file to the devbox directory.
func (d *Devbox) saveCfg() error {
	cfgPath := filepath.Join(d.srcDir, configFilename)
	return cuecfg.WriteFile(cfgPath, d.cfg)
}

func (d *Devbox) convertToPlan() *plansdk.Plan {
	configStages := []*Stage{d.cfg.InstallStage, d.cfg.BuildStage, d.cfg.StartStage}
	planStages := []*plansdk.Stage{{}, {}, {}}

	for i, stage := range configStages {
		if stage != nil {
			planStages[i] = &plansdk.Stage{
				Command: stage.Command,
			}
		}
	}
	return &plansdk.Plan{
		DevPackages:     d.cfg.Packages,
		RuntimePackages: d.cfg.Packages,
		InstallStage:    planStages[0],
		BuildStage:      planStages[1],
		StartStage:      planStages[2],
	}
}

func (d *Devbox) generateShellFiles() error {
	shellPlan, err := d.ShellPlan()
	if err != nil {
		return errors.WithStack(err)
	}
	if shellPlan.Invalid() {
		return shellPlan.Error()
	}
	return generate(d.srcDir, shellPlan, shellFiles)
}

func (d *Devbox) generateBuildFiles() error {
	buildPlan, err := d.BuildPlan()
	if err != nil {
		return errors.WithStack(err)
	}
	if buildPlan.Invalid() {
		return buildPlan.Error()
	}
	if buildPlan.Warning() != nil {
		fmt.Printf("[WARNING]: %s\n", buildPlan.Warning().Error())
	}
	return generate(d.srcDir, buildPlan, buildFiles)
}

func (d *Devbox) profileDir() string {
	return filepath.Join(d.srcDir, profileDir)
}

func (d *Devbox) profileBinDir() string {
	return filepath.Join(d.profileDir(), "bin")
}

func missingDevboxJSONError(dir string) error {

	// We try to prettify the `dir` before printing
	if dir == "." || dir == "" {
		dir = "this directory"
	} else {
		// Instead of a long absolute directory, print the relative directory

		wd, err := os.Getwd()
		// if an error occurs, then just use `dir`
		if err == nil {
			relDir, err := filepath.Rel(wd, dir)
			if err == nil {
				dir = relDir
			}
		}
	}
	return usererr.New("No devbox.json found in %s, or any parent directories. Did you run `devbox init` yet?", dir)
}

func findConfigDir(dir string) (string, error) {

	// Sanitize the directory and use the absolute path as canonical form
	cur, err := filepath.Abs(dir)
	if err != nil {
		return "", errors.WithStack(err)
	}

	for cur != "/" {
		debug.Log("finding %s in dir: %s\n", configFilename, cur)
		if plansdk.FileExists(filepath.Join(cur, configFilename)) {
			return cur, nil
		}
		cur = filepath.Dir(cur)
	}
	if plansdk.FileExists(filepath.Join(cur, configFilename)) {
		return cur, nil
	}
	return "", missingDevboxJSONError(dir)
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
	if err := d.ApplyDevNixDerivation(); err != nil {
		fmt.Println()
		return err
	}
	fmt.Println("done.")

	return nil
}

func (d *Devbox) printPackageUpdateMessage(mode installMode, pkgs []string) error {
	// (Only when in devbox shell) Prompt the user to run `hash -r` to ensure their
	// shell can access the most recently installed binaries, or ensure their
	// recently uninstalled binaries are not accidentally still available.
	if len(pkgs) > 0 && IsDevboxShellEnabled() {
		installedVerb := "installed"
		if mode == uninstall {
			installedVerb = "removed"
		}

		successMsg := fmt.Sprintf("%s is now %s.", pkgs[0], installedVerb)
		if len(pkgs) > 1 {
			successMsg = fmt.Sprintf("%s are now %s.", strings.Join(pkgs, ", "), installedVerb)
		}
		fmt.Fprint(d.writer, successMsg)
		fmt.Fprintln(d.writer, " Run `hash -r` to ensure your shell is updated.")
	}
	return nil
}

// ApplyDevNixDerivation ensures the local profile has exactly the packages in the development.nix file
//
// Will move to a store interface/package
func (d *Devbox) ApplyDevNixDerivation() error {

	cmdStr := fmt.Sprintf(
		"--profile %s --install -f %s/.devbox/gen/development.nix",
		filepath.Join(d.srcDir, profileDir),
		d.srcDir,
	)
	cmdParts := strings.Split(cmdStr, " ")
	execCmd := exec.Command("nix-env", cmdParts...)

	debug.Log("running command: %s\n", execCmd.Args)
	err := execCmd.Run()
	return errors.WithStack(err)
}

// Move to a utility package?
func IsDevboxShellEnabled() bool {
	inDevboxShell, err := strconv.ParseBool(os.Getenv("DEVBOX_SHELL_ENABLED"))
	if err != nil {
		return false
	}
	return inDevboxShell
}
