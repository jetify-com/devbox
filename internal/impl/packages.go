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

	pkgs, err := d.pendingPackagesForInstallation()
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
			nix.FlakeNixpkgs(d.cfg.Nixpkgs.Commit)+"#"+pkg,
		)
		cmd.Stdout = &nixPackageInstallWriter{d.writer}

		cmd.Env = nix.DefaultEnv()
		cmd.Stderr = cmd.Stdout
		err = cmd.Run()
		if err != nil {
			fmt.Fprintf(d.writer, "%s: ", stepMsg)
			color.New(color.FgRed).Fprintf(d.writer, "Fail\n")
			return errors.Wrapf(err, "Command: %s", cmd)
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
		return errors.Wrapf(err, "Command: %s", cmd)
	}
	fmt.Fprintf(d.writer, "Ensuring nixpkgs registry is downloaded: ")
	color.New(color.FgGreen).Fprintf(d.writer, "Success\n")
	return nil
}
