package impl

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/trace"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/plugin"
	"go.jetpack.io/devbox/internal/ux"
	"golang.org/x/exp/slices"
)

// packages.go has functions for adding, removing and getting info about nix packages

// Add adds the `pkgs` to the config (i.e. devbox.json) and nix profile for this devbox project
func (d *Devbox) Add(pkgs ...string) error {
	ctx, task := trace.NewTask(context.Background(), "devboxAdd")
	defer task.End()

	original := d.cfg.RawPackages
	// Check packages are valid before adding.
	for _, pkg := range pkgs {
		ok := nix.PkgExists(d.cfg.Nixpkgs.Commit, pkg)
		if !ok {
			return errors.WithMessage(nix.ErrPackageNotFound, pkg)
		}
	}

	// Add to Packages to config only if it's not already there
	for _, pkg := range pkgs {
		if slices.Contains(d.cfg.RawPackages, pkg) {
			continue
		}
		d.cfg.RawPackages = append(d.cfg.RawPackages, pkg)
	}
	if err := d.saveCfg(); err != nil {
		return err
	}

	d.pluginManager.ApplyOptions(plugin.WithAddMode())
	if err := d.ensurePackagesAreInstalled(ctx, install); err != nil {
		// if error installing, revert devbox.json
		// This is not perfect because there may be more than 1 package being
		// installed and we don't know which one failed. But it's better than
		// blindly add all packages.
		color.New(color.FgRed).Fprintf(
			d.writer,
			"There was an error installing nix packages: %v. "+
				"Packages were not added to devbox.json\n",
			strings.Join(pkgs, ", "),
		)
		d.cfg.RawPackages = original
		_ = d.saveCfg() // ignore error to ensure we return the original error
		return err
	}

	for _, pkg := range pkgs {
		if err := plugin.PrintReadme(
			pkg,
			d.projectDir,
			d.writer,
			false, /*markdown*/
		); err != nil {
			return err
		}
	}

	if IsDevboxShellEnabled() {
		plugin.PrintEnvUpdateMessage(d.projectDir, d.writer)
	}
	return nil
}

// Remove removes the `pkgs` from the config (i.e. devbox.json) and nix profile for this devbox project
func (d *Devbox) Remove(pkgs ...string) error {
	ctx, task := trace.NewTask(context.Background(), "devboxRemove")
	defer task.End()

	// First, save which packages are being uninstalled. Do this before we modify d.cfg.RawPackages below.
	uninstalledPackages := lo.Intersect(d.cfg.RawPackages, pkgs)

	var missingPkgs []string
	d.cfg.RawPackages, missingPkgs = lo.Difference(d.cfg.RawPackages, pkgs)

	if len(missingPkgs) > 0 {
		ux.Fwarning(
			d.writer,
			"the following packages were not found in your devbox.json: %s\n",
			strings.Join(missingPkgs, ", "),
		)
	}
	if err := d.saveCfg(); err != nil {
		return err
	}

	if err := plugin.Remove(d.projectDir, uninstalledPackages); err != nil {
		return err
	}

	if err := d.removePackagesFromProfile(ctx, uninstalledPackages); err != nil {
		return err
	}

	if err := d.ensurePackagesAreInstalled(ctx, uninstall); err != nil {
		return err
	}

	if IsDevboxShellEnabled() {
		plugin.PrintEnvUpdateMessage(d.projectDir, d.writer)
	}
	return nil
}

// installMode is an enum for helping with ensurePackagesAreInstalled implementation
type installMode string

const (
	install   installMode = "install"
	uninstall installMode = "uninstall"
	ensure    installMode = "ensure"
)

// ensurePackagesAreInstalled ensures that the nix profile has the packages specified
// in the config (devbox.json). The `mode` is used for user messaging to explain
// what operations are happening, because this function may take time to execute.
func (d *Devbox) ensurePackagesAreInstalled(ctx context.Context, mode installMode) error {
	defer trace.StartRegion(ctx, "ensurePackages").End()

	if err := d.generateShellFiles(); err != nil {
		return err
	}
	if mode == ensure {
		fmt.Fprintln(d.writer, "Ensuring packages are installed.")
	}

	if err := d.addPackagesToProfile(ctx, mode); err != nil {
		return err
	}

	return plugin.RemoveInvalidSymlinks(d.projectDir)
}

func (d *Devbox) profilePath() (string, error) {
	absPath := filepath.Join(d.projectDir, nix.ProfilePath)

	if err := resetProfileDirForFlakes(absPath); err != nil {
		debug.Log("ERROR: resetProfileDirForFlakes error: %v\n", err)
	}

	return absPath, errors.WithStack(os.MkdirAll(filepath.Dir(absPath), 0755))
}

// addPackagesToProfile inspects the packages in devbox.json, checks which of them
// are missing from the nix profile, and then installs each package individually into the
// nix profile.
func (d *Devbox) addPackagesToProfile(ctx context.Context, mode installMode) error {
	defer trace.StartRegion(ctx, "addNixProfilePkgs").End()

	if mode == uninstall {
		return nil
	}

	pkgs, err := d.pendingPackagesForInstallation(ctx)
	if err != nil {
		return err
	}

	if len(pkgs) == 0 {
		return nil
	}

	var msg string
	if len(pkgs) == 1 {
		msg = fmt.Sprintf("Installing package: %s.", pkgs[0])
	} else {
		msg = fmt.Sprintf("Installing %d packages: %s.", len(pkgs), strings.Join(pkgs, ", "))
	}
	fmt.Fprintf(d.writer, "\n%s\n\n", msg)

	profileDir, err := d.profilePath()
	if err != nil {
		return err
	}

	total := len(pkgs)
	for idx, pkg := range pkgs {
		stepNum := idx + 1

		stepMsg := fmt.Sprintf("[%d/%d] %s", stepNum, total, pkg)

		if err := nix.ProfileInstall(&nix.ProfileInstallArgs{
			CustomStepMessage: stepMsg,
			NixpkgsCommit:     d.cfg.Nixpkgs.Commit,
			Package:           pkg,
			ProfilePath:       profileDir,
			Writer:            d.writer,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (d *Devbox) removePackagesFromProfile(ctx context.Context, pkgs []string) error {
	defer trace.StartRegion(ctx, "removeNixProfilePkgs").End()

	profileDir, err := d.profilePath()
	if err != nil {
		return err
	}

	items, err := nix.ProfileListItems(d.writer, profileDir)
	if err != nil {
		return err
	}

	nameToAttributePath := map[string]string{}
	for _, item := range items {
		attrPath, err := item.AttributePath()
		if err != nil {
			return err
		}
		name, err := item.PackageName()
		if err != nil {
			return err
		}
		nameToAttributePath[name] = attrPath
	}

	for _, pkg := range pkgs {
		attrPath, ok := nameToAttributePath[pkg]
		if !ok {
			return errors.Errorf("Did not find AttributePath for package: %s", pkg)
		}

		// TODO: unify this with nix.ProfileRemove
		cmd := exec.Command("nix", "profile", "remove",
			"--profile", profileDir,
			attrPath,
		)
		cmd.Args = append(cmd.Args, nix.ExperimentalFlags()...)
		cmd.Stdout = d.writer
		cmd.Stderr = d.writer
		err = cmd.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

// pendingPackagesForInstallation returns a list of packages that are in
// devbox.json or global devbox.json but are not yet installed in the nix
// profile. It maintains the order of packages as specified by
// Devbox.packages() (higher priority first)
func (d *Devbox) pendingPackagesForInstallation(ctx context.Context) ([]string, error) {
	defer trace.StartRegion(ctx, "pendingPackages").End()

	profileDir, err := d.profilePath()
	if err != nil {
		return nil, err
	}

	items, err := nix.ProfileListItems(d.writer, profileDir)
	if err != nil {
		return nil, err
	}

	installed := map[string]bool{}
	for _, item := range items {
		packageName, err := item.PackageName()
		if err != nil {
			return nil, err
		}
		installed[packageName] = true
	}

	pending := []string{}
	for _, pkg := range d.mergedPackages() {
		if _, ok := installed[pkg]; !ok {
			pending = append(pending, pkg)
		}
	}
	return pending, nil
}

var resetCheckDone = false

// resetProfileDirForFlakes ensures the profileDir directory is cleared of old
// state if the Flakes feature has been changed, from the previous execution of a devbox command.
func resetProfileDirForFlakes(profileDir string) (err error) {
	if resetCheckDone {
		return nil
	}
	defer func() {
		if err == nil {
			resetCheckDone = true
		}
	}()

	dir, err := filepath.EvalSymlinks(profileDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return errors.WithStack(err)
	}

	// older nix profiles have a manifest.nix file present
	_, err = os.Stat(filepath.Join(dir, "manifest.nix"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return errors.WithStack(err)
	}

	return errors.WithStack(os.Remove(profileDir))
}
