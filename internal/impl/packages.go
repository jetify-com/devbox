package impl

import (
	"fmt"
	"os/exec"
	"strconv"
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

	items, err := d.nixProfileListItems()
	if err != nil {
		return err
	}

	nameToAttributePath := map[string]string{}
	for _, item := range items {
		attrPath, err := item.attributePath()
		if err != nil {
			return err
		}
		packageName, err := item.packageName()
		if err != nil {
			return err
		}
		nameToAttributePath[packageName] = attrPath
	}

	profileDir, err := d.profileDir()
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		attrPath, ok := nameToAttributePath[pkg]
		if !ok {
			return errors.Errorf("Did not find attributePath for package: %s", pkg)
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

	items, err := d.nixProfileListItems()
	if err != nil {
		return nil, err
	}

	installed := map[string]bool{}
	for _, item := range items {
		packageName, err := item.packageName()
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

// nixProfileListItems returns a list of the installed packages
func (d *Devbox) nixProfileListItems() ([]*nixProfileListItem, error) {
	if featureflag.Flakes.Disabled() {
		return nil, errors.New("Not supported for legacy non-flakes implementation")
	}

	profileDir, err := d.profileDir()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	cmd := exec.Command(
		"nix", "profile", "list",
		"--extra-experimental-features", "nix-command flakes",
		"--profile", profileDir)

	// We set stderr to a different output than stdout
	// to ensure error output is not mingled with the stdout output
	// that we need to parse.
	cmd.Stderr = d.writer

	out, err := cmd.Output()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// The `out` output is of the form:
	// <index> <UnlockedReference> <LockedReference> <NixStorePath>
	//
	// Using an example:
	// 0 github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19 github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19 /nix/store/w0lyimyyxxfl3gw40n46rpn1yjrl3q85-go-1.19.3
	// 1 github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.vim github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.vim /nix/store/gapbqxx1d49077jk8ay38z11wgr12p23-vim-9.0.0609
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	items := []*nixProfileListItem{}
	for _, line := range lines {
		item, err := parseNixProfileListItemIfAny(line)
		if err != nil {
			return nil, err
		}
		if item == nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

// nixProfileListItem is a go-struct of a line of printed output from `nix profile list`
type nixProfileListItem struct {
	// An integer that can be used to unambiguously identify the package in
	// invocations of nix profile remove and nix profile upgrade.
	index int

	// The original ("unlocked") flake reference and output attribute path used at installation time.
	unlockedReference string

	// The locked flake reference to which the unlocked flake reference was resolved.
	lockedReference string

	// The store path(s) of the package.
	nixStorePath string
}

func parseNixProfileListItemIfAny(line string) (*nixProfileListItem, error) {
	// line is a line of printed output from `nix profile list`
	//
	// line example:
	// 0 github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19 github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19 /nix/store/w0lyimyyxxfl3gw40n46rpn1yjrl3q85-go-1.19.3
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}

	// parts example:
	// [
	//   0
	//   github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19
	//   github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19
	//   /nix/store/w0lyimyyxxfl3gw40n46rpn1yjrl3q85-go-1.19.3
	// ]
	parts := strings.Split(line, " ")
	if len(parts) != 4 {
		return nil, errors.Errorf(
			"Expected 4 parts for line in nix profile list, but got %d parts. Line: %s",
			len(parts),
			line,
		)
	}

	index, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &nixProfileListItem{
		index:             index,
		unlockedReference: parts[1],
		lockedReference:   parts[2],
		nixStorePath:      parts[3],
	}, nil
}

// attributePath parses the package attribute from the nixProfileListItem.lockedReference
//
// For example:
// if nixProfileListItem.lockedReference = github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19
// then attributePath = legacyPackages.x86_64-darwin.go_1_19
func (item *nixProfileListItem) attributePath() (string, error) {

	// lockedReference example:
	// github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19

	// attributePath example:
	// legacyPackages.x86_64.go_1_19
	_ /*nixpkgs*/, attrPath, found := strings.Cut(item.lockedReference, "#")
	if !found {
		return "", errors.Errorf(
			"expected to find # in lockedReference: %s from nixProfileListItem: %s",
			item.lockedReference,
			item.String(),
		)
	}
	return attrPath, nil
}

// packageName parses the package name from the nixProfileListItem.lockedReference
//
// For example:
// if nixProfileListItem.lockedReference = github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19
// then attributePath = legacyPackages.x86_64-darwin.go_1_19
// and then packageName = go_1_19
func (item *nixProfileListItem) packageName() (string, error) {
	attrPath, err := item.attributePath()
	if err != nil {
		return "", err
	}

	parts := strings.Split(attrPath, ".")
	if len(parts) < 2 {
		return "", errors.Errorf(
			"Expected >= 2 parts for attributePath in nix profile list, but got %d parts. attributePath: %s",
			len(parts),
			attrPath,
		)
	}

	packageName := parts[len(parts)-1]
	return packageName, nil
}

// String serializes the nixProfileListItem back into the format printed by `nix profile list`
func (item *nixProfileListItem) String() string {
	return fmt.Sprintf("%d %s %s %s",
		item.index,
		item.unlockedReference,
		item.lockedReference,
		item.nixStorePath,
	)
}
