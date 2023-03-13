package impl

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/trace"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/fileutil"
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

	return d.printPackageUpdateMessage(install, pkgs)
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

	return d.printPackageUpdateMessage(uninstall, uninstalledPackages)
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

	if featureflag.Flakes.Enabled() {
		if err := d.addPackagesToProfile(ctx, mode); err != nil {
			return err
		}

	} else {
		if mode == install || mode == uninstall {
			installingVerb := "Installing"
			if mode == uninstall {
				installingVerb = "Uninstalling"
			}
			_, _ = fmt.Fprintf(d.writer, "%s nix packages.\n", installingVerb)
		}

		// We need to re-install the packages
		if err := d.installNixProfile(ctx); err != nil {
			fmt.Fprintln(d.writer)
			return errors.Wrap(err, "apply Nix derivation")
		}
	}

	return plugin.RemoveInvalidSymlinks(d.projectDir)
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

		successMsg := fmt.Sprintf("%s (%s) is now %s.\n", pkgs[0], infos[0], verb)
		if len(pkgs) > 1 {
			pkgsWithVersion := []string{}
			for idx, pkg := range pkgs {
				pkgsWithVersion = append(
					pkgsWithVersion,
					fmt.Sprintf("%s (%s)", pkg, infos[idx]),
				)
			}
			successMsg = fmt.Sprintf(
				"%s are now %s.\n",
				strings.Join(pkgsWithVersion, ", "),
				verb,
			)
		}
		fmt.Fprint(d.writer, successMsg)

		// (Only when in devbox shell) Prompt the user to run hash -r
		// to ensure we refresh the shell hash and load the proper environment.
		if IsDevboxShellEnabled() {
			if err := plugin.PrintEnvUpdateMessage(
				lo.Ternary(mode == install, pkgs, []string{}),
				d.projectDir,
				d.writer,
			); err != nil {
				return err
			}
		}
	} else {
		fmt.Fprintf(d.writer, "No packages %s.\n", verb)
	}
	return nil
}

// installNixProfile installs or uninstalls packages to or from this
// devbox's Nix profile so that it matches what's in development.nix
func (d *Devbox) installNixProfile(ctx context.Context) (err error) {
	defer trace.StartRegion(ctx, "installNixProfile").End()

	profileDir, err := d.profilePath()
	if err != nil {
		return err
	}

	cmd := exec.Command(
		"nix-env",
		"--profile", profileDir,
		"--install",
		"-f", filepath.Join(d.projectDir, ".devbox/gen/development.nix"),
	)

	cmd.Stdout = &nix.PackageInstallWriter{Writer: d.writer}

	cmd.Stderr = cmd.Stdout

	err = cmd.Run()

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return errors.Errorf(
			"running command %s: exit status %d with command stderr: %s",
			cmd, exitErr.ExitCode(), string(exitErr.Stderr),
		)
	}
	if err != nil {
		return errors.Errorf("running command %s: %v", cmd, err)
	}
	return nil
}

func (d *Devbox) profilePath() (string, error) {
	absPath := filepath.Join(d.projectDir, nix.ProfilePath)

	if err := resetProfileDirForFlakes(absPath); err != nil {
		debug.Log("ERROR: resetProfileDirForFlakes error: %v\n", err)
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return "", errors.WithStack(err)
	}

	return absPath, nil
}

// addPackagesToProfile inspects the packages in devbox.json, checks which of them
// are missing from the nix profile, and then installs each package individually into the
// nix profile.
func (d *Devbox) addPackagesToProfile(ctx context.Context, mode installMode) error {
	defer trace.StartRegion(ctx, "addNixProfilePkgs").End()

	if featureflag.Flakes.Disabled() {
		return nil
	}
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

	// Packages with higher priority number i.e. lower actual priority
	// are to be installed first, so that any conflicts are resolved in favor
	// of later packages (having lower priority number i.e. higher actual priority)
	//
	// We use stable sort so that users can manually change the order of packages
	// in their configs, if they have particular opinions about which package should
	// win any conflicts.
	sort.SliceStable(pkgs, func(i, j int) bool {
		return d.getPackagePriority(pkgs[i]) > d.getPackagePriority(pkgs[j])
	})

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
			Priority:          d.getPackagePriority(pkg),
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

	if !featureflag.Flakes.Enabled() {
		return nil
	}

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

func (d *Devbox) pendingPackagesForInstallation(ctx context.Context) ([]string, error) {
	defer trace.StartRegion(ctx, "pendingPackages").End()

	if featureflag.Flakes.Disabled() {
		return nil, errors.New("Not implemented for legacy non-flakes devbox")
	}

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
	for _, pkg := range d.packages() {
		if _, ok := installed[pkg]; !ok {
			pending = append(pending, pkg)
		}
	}
	return pending, nil
}

// This sets the priority of non-devbox.json packages to be slightly lower (higher number)
// than devbox.json packages. This matters for profile installs, but doesn't matter
// much for the flakes.nix file. There we rely on the order of packages (local ahead of global)
func (d *Devbox) getPackagePriority(pkg string) string {
	for _, p := range d.cfg.RawPackages {
		if p == pkg {
			return "5"
		}
	}
	return "6" // Anything higher than 5 (default) would be correct
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
	if err != nil {
		return errors.WithStack(err)
	}

	needsReset := false
	if featureflag.Flakes.Enabled() {
		// older nix profiles have a manifest.nix file present
		needsReset = fileutil.Exists(filepath.Join(dir, "manifest.nix"))
	} else {
		// newer flake nix profiles have a manifest.json file present
		needsReset = fileutil.Exists(filepath.Join(dir, "manifest.json"))
	}

	if !needsReset {
		return nil
	}

	return errors.WithStack(os.Remove(profileDir))
}
