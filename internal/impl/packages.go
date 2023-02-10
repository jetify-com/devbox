package impl

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/fileutil"
	"go.jetpack.io/devbox/internal/nix"
)

// packages.go has functions for adding, removing and getting info about nix packages

func (d *Devbox) profileDir() (string, error) {
	absPath := filepath.Join(d.projectDir, nix.ProfilePath)

	if err := resetProfileDirForFlakes(absPath); err != nil {
		debug.Log("ERROR: resetProfileDirForFlakes error: %v\n", err)
	}

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

	pkgs, err := d.pendingPackagesForInstallation()
	if err != nil {
		return err
	}

	if len(pkgs) == 0 {
		return nil
	}

	var msg string
	if len(pkgs) == 1 {
		msg = fmt.Sprintf("Installing the following package: %s.\n", pkgs[0])
	} else {
		msg = fmt.Sprintf("Installing the following %d packages: %s.\n", len(pkgs), strings.Join(pkgs, ", "))
	}
	color.New(color.FgGreen).Fprintf(d.writer, msg)

	profileDir, err := d.profileDir()
	if err != nil {
		return err
	}

	// Append an empty string to prefetch the nixpkgs as a distinct step
	// TODO savil. Its a bit odd to always show nixpkgs as being installed during `devbox add`.
	// Can we inspect the nix store to check if its been pre-fetched already?
	packages := append([]string{""}, pkgs...)

	total := len(packages)
	for idx, pkg := range packages {
		stepNum := idx + 1

		var stepMsg string
		if pkg == "" {
			stepMsg = fmt.Sprintf("[%d/%d] nixpkgs registry", stepNum, total)
		} else {
			stepMsg = fmt.Sprintf("[%d/%d] %s", stepNum, total, pkg)
		}
		fmt.Printf("%s\n", stepMsg)

		var cmd *exec.Cmd
		if pkg == "" {
			cmd = exec.Command(
				"nix", "flake", "prefetch",
				"--extra-experimental-features", "nix-command flakes",
				nix.FlakeNixpkgs(d.cfg.Nixpkgs.Commit),
			)
		} else {
			cmd = exec.Command(
				"nix", "profile", "install",
				"--profile", profileDir,
				"--extra-experimental-features", "nix-command flakes",
				nix.FlakeNixpkgs(d.cfg.Nixpkgs.Commit)+"#"+pkg,
			)
		}

		cmd.Env = nix.DefaultEnv()
		cmd.Stdout = &nixPackageInstallWriter{d.writer}
		cmd.Stderr = cmd.Stdout
		err = cmd.Run()

		if err != nil {

			// ExitErrors can give us more information so handle that specially.
			var errorMsg string
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				errorMsg = fmt.Sprintf(
					"Error running command %s. Exit status is %d. Command stderr: %s",
					cmd, exitErr.ExitCode(), string(exitErr.Stderr),
				)
			} else {
				errorMsg = fmt.Sprintf("Error running command %s. Error: %v", cmd, err)
			}
			fmt.Fprint(d.writer, errorMsg)

			fmt.Fprintf(d.writer, "%s: ", stepMsg)
			color.New(color.FgRed).Fprintf(d.writer, "Fail\n")

			return errors.New(errorMsg)
		}

		fmt.Fprintf(d.writer, "%s: ", stepMsg)
		color.New(color.FgGreen).Fprintf(d.writer, "Success\n")
	}

	return nil
}

func (d *Devbox) removePackagesFromProfile(pkgs []string) error {
	if !featureflag.Flakes.Enabled() {
		return nil
	}

	profileDir, err := d.profileDir()
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

		cmd := exec.Command("nix", "profile", "remove",
			"--profile", profileDir,
			"--extra-experimental-features", "nix-command flakes",
			attrPath,
		)
		cmd.Stdout = d.writer
		cmd.Stderr = d.writer
		err = cmd.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Devbox) pendingPackagesForInstallation() ([]string, error) {
	if featureflag.Flakes.Disabled() {
		return nil, errors.New("Not implemented for legacy non-flakes devbox")
	}

	profileDir, err := d.profileDir()
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
	for _, pkg := range d.cfg.Packages {
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
