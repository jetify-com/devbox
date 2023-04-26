package impl

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/trace"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"

	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/lockfile"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/plugin"
	"go.jetpack.io/devbox/internal/searcher"
	"go.jetpack.io/devbox/internal/ux"
	"go.jetpack.io/devbox/internal/wrapnix"
)

// packages.go has functions for adding, removing and getting info about nix packages

// Add adds the `pkgs` to the config (i.e. devbox.json) and nix profile for this devbox project
func (d *Devbox) Add(ctx context.Context, pkgs ...string) error {
	ctx, task := trace.NewTask(ctx, "devboxAdd")
	defer task.End()

	pkgs = lo.Uniq(pkgs)

	pkgs, err := searcher.GenLockedReferences(pkgs)
	if err != nil {
		return err
	}

	original := d.cfg.Packages
	// Check packages are valid before adding.
	for _, pkg := range pkgs {
		ok, err := nix.PkgExists(d.cfg.Nixpkgs.Commit, pkg, d.projectDir)
		if err != nil {
			return err
		}
		if !ok {
			return errors.WithMessage(nix.ErrPackageNotFound, pkg)
		}
	}

	// Add to Packages of the config only if it's not already there
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
	if err := d.ensurePackagesAreInstalled(ctx, install); err != nil {
		// if installation fails, revert devbox.json
		// This is not perfect because there may be more than 1 package being
		// installed and we don't know which one failed. But it's better than
		// blindly add all packages.
		color.New(color.FgRed).Fprintf(
			d.writer,
			"There was an error installing nix packages: %v. "+
				"Packages were not added to devbox.json\n",
			strings.Join(pkgs, ", "),
		)
		d.cfg.Packages = original
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

	return wrapnix.CreateWrappers(ctx, d)
}

// Remove removes the `pkgs` from the config (i.e. devbox.json) and nix profile for this devbox project
func (d *Devbox) Remove(ctx context.Context, pkgs ...string) error {
	ctx, task := trace.NewTask(ctx, "devboxRemove")
	defer task.End()

	pkgs = lo.Uniq(pkgs)

	// First, save which packages are being uninstalled. Do this before we modify d.cfg.RawPackages below.
	uninstalledPackages := lo.Intersect(d.cfg.Packages, pkgs)
	remainingPkgs, missingPkgs := lo.Difference(d.cfg.Packages, pkgs)
	if len(missingPkgs) > 0 {
		ux.Fwarning(
			d.writer,
			"the following packages were not found in your devbox.json: %s\n",
			strings.Join(missingPkgs, ", "),
		)
	}
	d.cfg.Packages = remainingPkgs
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

	return wrapnix.CreateWrappers(ctx, d)
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

	lock, err := lockfile.Local(d)
	if err != nil {
		return err
	}

	upToDate, err := lock.IsUpToDate()
	if err != nil {
		return err
	}
	if upToDate {
		return nil
	}

	if err := d.generateShellFiles(); err != nil {
		return err
	}
	if mode == ensure {
		fmt.Fprintln(d.writer, "Ensuring packages are installed.")
	}

	if err := d.addPackagesToProfile(ctx, mode); err != nil {
		return err
	}

	if err := plugin.RemoveInvalidSymlinks(d.projectDir); err != nil {
		return err
	}

	// Force print-dev-env cache to be recomputed.
	if _, err = d.computeNixEnv(ctx, false /*use cache*/); err != nil {
		return err
	}

	return lock.Update()
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
			ProjectDir:        d.projectDir,
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

	for _, pkg := range pkgs {
		index, err := nix.ProfileListIndex(&nix.ProfileListIndexArgs{
			Writer:     d.writer,
			Pkg:        pkg,
			ProjectDir: d.projectDir,
			ProfileDir: profileDir,
		})
		if err != nil {
			ux.Ferror(d.writer, "Package %s not found in profile. Skipping.\n", pkg)
			continue
		}

		// TODO: unify this with nix.ProfileRemove
		cmd := exec.Command("nix", "profile", "remove",
			"--profile", profileDir,
			fmt.Sprintf("%d", index),
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

	pending := []string{}
	list, err := nix.ProfileListItems(d.writer, profileDir)
	if err != nil {
		return nil, err
	}
	for _, pkg := range d.packages() {
		_, err := nix.ProfileListIndex(&nix.ProfileListIndexArgs{
			List:       list,
			Writer:     d.writer,
			Pkg:        pkg,
			ProjectDir: d.projectDir,
			ProfileDir: profileDir,
		})
		if err != nil {
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
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return errors.WithStack(err)
	}

	// older nix profiles have a manifest.nix file present
	_, err = os.Stat(filepath.Join(dir, "manifest.nix"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return errors.WithStack(err)
	}

	return errors.WithStack(os.Remove(profileDir))
}
