package impl

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/nix"
)

// packages.go has functions for adding, removing and getting info about nix packages

// addPackagesToProfile inspects the packages in devbox.json, and checks which of them
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
	} else {
		var msg string
		if len(pkgs) == 1 {
			msg = fmt.Sprintf("Installing the following package: %s.\n", pkgs[0])
		} else {
			msg = fmt.Sprintf("Installing the following %d packages: %s.\n", len(pkgs), strings.Join(pkgs, ", "))
		}
		color.New(color.FgGreen).Fprintf(d.writer, msg)
	}

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

		var msg string
		if pkg == "" {
			msg = fmt.Sprintf("[%d/%d] nixpkgs registry", stepNum, total)
		} else {
			msg = fmt.Sprintf("[%d/%d] %s", stepNum, total, pkg)
		}
		fmt.Printf("%s\n", msg)

		var cmd *exec.Cmd
		if pkg != "" {
			cmd = exec.Command(
				"nix", "profile", "install",
				"--profile", profileDir,
				"--extra-experimental-features", "nix-command flakes",
				nix.FlakeNixpkgs(d.cfg.Nixpkgs.Commit)+"#"+pkg,
			)
		} else {
			cmd = exec.Command(
				"nix", "flake", "prefetch",
				"--extra-experimental-features", "nix-command flakes",
				nix.FlakeNixpkgs(d.cfg.Nixpkgs.Commit),
			)
		}

		cmd.Env = nix.DefaultEnv()
		cmd.Stdout = &nixPackageInstallWriter{d.writer}
		cmd.Stderr = cmd.Stdout
		err = cmd.Run()
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			msg := fmt.Sprintf(
				"running command %s: exit status %d with command stderr: %s",
				cmd, exitErr.ExitCode(), string(exitErr.Stderr),
			)
			color.New(color.FgRed).Printf("%s: Fail\n", msg)
			return errors.New(msg)
		}
		if err != nil {
			msg := fmt.Sprintf("running command %s: %v", cmd, err)
			color.New(color.FgRed).Printf("%s: Fail\n", msg)
			return errors.New(msg)
		}
		fmt.Printf("%s: ", msg)
		color.New(color.FgGreen).Printf("Success\n")
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
		packageName, err := item.PackageName()
		if err != nil {
			return err
		}
		nameToAttributePath[packageName] = attrPath
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
