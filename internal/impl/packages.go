package impl

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/fileutil"
	"go.jetpack.io/devbox/internal/nix"
)

// packages.go has functions for adding, removing and getting info about nix packages

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

func (d *Devbox) profileBinPath() (string, error) {
	profileDir, err := d.profilePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(profileDir, "bin"), nil
}

// addPackagesToProfile inspects the packages in devbox.json, checks which of them
// are missing from the nix profile, and then installs each package individually into the
// nix profile.
func (d *Devbox) addPackagesToProfile(mode installMode) error {
	if featureflag.Flakes.Disabled() {
		return nil
	}
	if mode == uninstall {
		return nil
	}

	if err := d.ensureNixpkgsPrefetched(); err != nil {
		return err
	}

	pkgs, err := d.pendingPackagesForInstallation(mode)
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
		fmt.Printf("%s\n", stepMsg)

		cmd := exec.Command(
			"nix", "profile", "install",
			"--profile", profileDir,
			"--extra-experimental-features", "nix-command flakes",
		)

		if isPhpRelatedPackage(pkg) {
			cmd.Args = append(cmd.Args, fmt.Sprintf(".devbox/gen/flake/php/flake.nix#%s", pkg))
		} else {
			cmd.Args = append(cmd.Args, nix.FlakeNixpkgs(d.cfg.Nixpkgs.Commit)+"#"+pkg)
		}
		cmd.Stdout = &nixPackageInstallWriter{d.writer}

		cmd.Env = nix.DefaultEnv()
		cmd.Stderr = cmd.Stdout
		err = cmd.Run()
		if err != nil {
			fmt.Fprintf(d.writer, "%s: ", stepMsg)
			color.New(color.FgRed).Fprintf(d.writer, "Fail\n")

			return errors.New(commandErrorMessage(cmd, err))
		}

		fmt.Fprintf(d.writer, "%s: ", stepMsg)
		color.New(color.FgGreen).Fprintf(d.writer, "Success\n")
	}

	// Add a newline
	fmt.Println()

	return nil
}

func (d *Devbox) removePackagesFromProfile(pkgs []string, mode installMode) error {
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

	nameToStorePath := map[string]string{}
	for _, item := range items {
		name, err := item.PackageName()
		if err != nil {
			return err
		}
		nameToStorePath[name] = item.StorePath()
	}

	for _, pkg := range pkgs {

		storePath, ok := nameToStorePath[pkg]
		if !ok {
			return errors.Errorf("Did not find StorePath for package: %s", pkg)
		}

		cmd := exec.Command("nix", "profile", "remove",
			"--profile", profileDir,
			"--extra-experimental-features", "nix-command flakes",
			storePath,
		)
		cmd.Stdout = d.writer
		cmd.Stderr = d.writer
		err = cmd.Run()
		if err != nil {
			return err
		}
	}

	// If any of the packages left are php-related, then we need to ensure they are reapplied.
	// We apply the mode-is-uninstall check to avoid recursion loops because this function (removePackagesFromProfile)
	// is also called when ensurePackagesAreInstalled for mode in {ensure, install, phpReinstall}.
	if mode == uninstall && hasPhpRelatedPackage(pkgs) {
		if err := d.ensurePackagesAreInstalled(phpReinstall); err != nil {
			return err
		}
	}

	return nil
}

func (d *Devbox) pendingPackagesForInstallation(mode installMode) ([]string, error) {
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

	// If there is any phpPackagePending, or mode is PhpReinstall,
	// then we must mark all php packages as pending.
	// This will re-install them by the caller (presumably).
	// Reason: php extensions may require php and related packages to be recompiled.
	if mode == phpReinstall || hasPhpRelatedPackage(pending) {
		for _, pkg := range d.packages() {
			if isPhpRelatedPackage(pkg) {
				pending = append(pending, pkg)
			}
		}

		// Alas, to avoid nix profile priority conflicts,
		// we must remove the php packages from the profile.
		installedPhpPackages := lo.Filter(d.packages(), func(pkg string, _ int) bool {
			_, isInstalled := installed[pkg]
			return isPhpRelatedPackage(pkg) && isInstalled
		})
		if len(installedPhpPackages) > 0 {
			color.New(color.FgHiYellow).Fprint(d.writer, "PHP packages will need to be re-installed.\n")
			if err := d.removePackagesFromProfile(installedPhpPackages, mode); err != nil {
				return nil, err
			}
		}
	}

	// De-duplicate entries.
	pending = lo.Uniq(pending)

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

// ensureNixpkgsPrefetched runs the prefetch step to download the flake of the registry
func (d *Devbox) ensureNixpkgsPrefetched() error {
	fmt.Fprintf(d.writer, "Ensuring nixpkgs registry is downloaded.\n")
	cmd := exec.Command(
		"nix", "flake", "prefetch",
		"--extra-experimental-features", "nix-command flakes",
		nix.FlakeNixpkgs(d.cfg.Nixpkgs.Commit),
	)
	cmd.Env = nix.DefaultEnv()
	cmd.Stdout = d.writer
	cmd.Stderr = cmd.Stdout
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(d.writer, "Ensuring nixpkgs registry is downloaded: ")
		color.New(color.FgRed).Fprintf(d.writer, "Fail\n")
		return errors.New(commandErrorMessage(cmd, err))
	}
	fmt.Fprintf(d.writer, "Ensuring nixpkgs registry is downloaded: ")
	color.New(color.FgGreen).Fprintf(d.writer, "Success\n")
	return nil
}

// Consider moving to cobra middleware where this could be generalized. There is
// a complication in that its current form is useful because of the exec.Cmd. This
// would be missing in the middleware, unless we pass it along by wrapping the error in
// another struct.
func commandErrorMessage(cmd *exec.Cmd, err error) string {
	var errorMsg string

	// ExitErrors can give us more information so handle that specially.
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		errorMsg = fmt.Sprintf(
			"Error running command %s. Exit status is %d. Command stderr: %s",
			cmd, exitErr.ExitCode(), string(exitErr.Stderr),
		)
	} else {
		errorMsg = fmt.Sprintf("Error running command %s. Error: %v", cmd, err)
	}
	return errorMsg
}

func isPhpRelatedPackage(pkg string) bool {
	return strings.HasPrefix(pkg, "php")
}

func hasPhpRelatedPackage(pkgs []string) bool {
	return lo.SomeBy(pkgs, func(pkg string) bool { return isPhpRelatedPackage(pkg) })
}
