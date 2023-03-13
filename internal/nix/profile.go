package nix

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/debug"
)

// ProfileListItems returns a list of the installed packages
func ProfileListItems(writer io.Writer, profileDir string) ([]*NixProfileListItem, error) {
	if featureflag.Flakes.Disabled() {
		return nil, errors.New("Not supported for legacy non-flakes implementation")
	}

	cmd := exec.Command(
		"nix", "profile", "list",
		"--profile", profileDir,
	)
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)

	// We set stderr to a different output than stdout
	// to ensure error output is not mingled with the stdout output
	// that we need to parse.
	cmd.Stderr = writer

	// The `out` output is of the form:
	// <index> <UnlockedReference> <LockedReference> <NixStorePath>
	//
	// Using an example:
	// 0 github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19 github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19 /nix/store/w0lyimyyxxfl3gw40n46rpn1yjrl3q85-go-1.19.3
	// 1 github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.vim github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.vim /nix/store/gapbqxx1d49077jk8ay38z11wgr12p23-vim-9.0.0609

	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if err := cmd.Start(); err != nil {
		return nil, errors.WithStack(err)
	}

	items := []*NixProfileListItem{}
	scanner := bufio.NewScanner(out)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		item, err := parseNixProfileListItem(line)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := cmd.Wait(); err != nil {
		return nil, errors.WithStack(err)
	}
	return items, nil
}

// NixProfileListItem is a go-struct of a line of printed output from `nix profile list`
// docs: https://nixos.org/manual/nix/stable/command-ref/new-cli/nix3-profile-list.html
type NixProfileListItem struct {
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

// parseNixProfileListItem reads each line of output (from `nix profile list`) and converts
// into a golang struct. Refer to NixProfileListItem struct definition for explanation of each field.
func parseNixProfileListItem(line string) (*NixProfileListItem, error) {

	scanner := bufio.NewScanner(strings.NewReader(line))
	scanner.Split(bufio.ScanWords)

	if !scanner.Scan() {
		return nil, errors.New("incomplete nix profile list line. Expected index.")
	}
	index, err := strconv.Atoi(scanner.Text())
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if !scanner.Scan() {
		return nil, errors.New("incomplete nix profile list line. Expected unlockedReference.")
	}
	unlockedReference := scanner.Text()

	if !scanner.Scan() {
		return nil, errors.New("incomplete nix profile list line. Expected lockedReference")
	}
	lockedReference := scanner.Text()

	if !scanner.Scan() {
		return nil, errors.New("incomplete nix profile list line. Expected nixStorePath.")
	}
	nixStorePath := scanner.Text()

	return &NixProfileListItem{
		index:             index,
		unlockedReference: unlockedReference,
		lockedReference:   lockedReference,
		nixStorePath:      nixStorePath,
	}, nil
}

// AttributePath parses the package attribute from the NixProfileListItem.lockedReference
//
// For example:
// if NixProfileListItem.lockedReference = github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19
// then AttributePath = legacyPackages.x86_64-darwin.go_1_19
func (item *NixProfileListItem) AttributePath() (string, error) {

	// lockedReference example:
	// github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19

	// AttributePath example:
	// legacyPackages.x86_64.go_1_19
	_ /*nixpkgs*/, attrPath, found := strings.Cut(item.lockedReference, "#")
	if !found {
		return "", errors.Errorf(
			"expected to find # in lockedReference: %s from NixProfileListItem: %s",
			item.lockedReference,
			item.String(),
		)
	}
	return attrPath, nil
}

// PackageName parses the package name from the NixProfileListItem.lockedReference
//
// For example:
// if NixProfileListItem.lockedReference = github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19
// then AttributePath = legacyPackages.x86_64-darwin.go_1_19
// and then PackageName = go_1_19
func (item *NixProfileListItem) PackageName() (string, error) {
	attrPath, err := item.AttributePath()
	if err != nil {
		return "", err
	}

	parts := strings.Split(attrPath, ".")
	if len(parts) < 2 {
		return "", errors.Errorf(
			"Expected >= 2 parts for AttributePath in nix profile list, but got %d parts. AttributePath: %s",
			len(parts),
			attrPath,
		)
	}

	packageName := strings.Join(parts[2:], ".")
	return packageName, nil
}

// String serializes the NixProfileListItem back into the format printed by `nix profile list`
func (item *NixProfileListItem) String() string {
	return fmt.Sprintf("%d %s %s %s",
		item.index,
		item.unlockedReference,
		item.lockedReference,
		item.nixStorePath,
	)
}

type ProfileInstallArgs struct {
	CustomStepMessage string
	Priority          string
	ExtraFlags        []string
	NixpkgsCommit     string
	Package           string
	ProfilePath       string
	Writer            io.Writer
}

// ProfileInstall calls nix profile install with default profile
func ProfileInstall(args *ProfileInstallArgs) error {
	if err := ensureNixpkgsPrefetched(args.Writer, args.NixpkgsCommit); err != nil {
		return err
	}
	stepMsg := args.Package
	if args.CustomStepMessage != "" {
		stepMsg = args.CustomStepMessage
		// Only print this first one if we have a custom message. Otherwise it feels
		// repetitive.
		fmt.Fprintf(args.Writer, "%s\n", stepMsg)
	}

	err := profileInstall(args)
	if err != nil {
		if errors.Is(err, ErrPackageNotFound) {
			return err
		}

		fmt.Fprintf(args.Writer, "%s: ", stepMsg)
		color.New(color.FgRed).Fprintf(args.Writer, "Fail\n")
		return err
	}

	fmt.Fprintf(args.Writer, "%s: ", stepMsg)
	color.New(color.FgGreen).Fprintf(args.Writer, "Success\n")

	return nil
}

func profileInstall(args *ProfileInstallArgs) error {

	cmd := exec.Command(
		"nix", "profile", "install",
		"--profile", args.ProfilePath,
		"--impure", // for NIXPKGS_ALLOW_UNFREE
		FlakeNixpkgs(args.NixpkgsCommit)+"#"+args.Package,
	)
	cmd.Env = AllowUnfreeEnv()
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)
	if args.Priority != "" {
		cmd.Args = append(cmd.Args, "--priority", args.Priority)
	}
	cmd.Args = append(cmd.Args, args.ExtraFlags...)
	writer := &PackageInstallWriter{args.Writer}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = io.MultiWriter(&stdout, writer)
	cmd.Stderr = io.MultiWriter(&stderr, writer)

	if err := cmd.Run(); err != nil {
		if strings.Contains(stderr.String(), "does not provide attribute") {
			return ErrPackageNotFound
		}

		// If two packages being installed seek to install two nix packages (could be themselves,
		// or could be a dependency) that have the same binary name,
		// then `nix profile install` will fail with `conflicting packages` error (as of nix version 2.14.0)
		//
		// An example error message looks like the following:
		// error: files '/nix/store/spgr12gk13af8flz7akbs18fj4whqss2-bundler-2.4.5/bin/bundle' and
		// '/nix/store/l4wmx8lfn6hlcfmbyhmksm024f8hixm1-ruby-3.1.2/bin/bundle' have the same priority 5;
		// use 'nix-env --set-flag priority NUMBER INSTALLED_PKGNAME' or type 'nix profile install --help'
		// if using 'nix profile' to find out howto change the priority of one of the conflicting packages
		// (0 being the highest priority)
		//
		// However, for the purposes of starting a shell with these packages, nix flakes will give
		// precedence to the later package. We enable similar functionality by increasing the priority
		// of any package being installed and conflicting with a previously installed package: the
		// package being installed later "wins".
		isConflictingPackagesError := strings.Contains(stdout.String(), "conflicting packages") ||
			strings.Contains(stderr.String(), "conflicting packages")
		if isConflictingPackagesError && args.Priority != "0" {

			priority := args.Priority
			if priority == "" {
				priority = "5" // 5 is the default priority in `nix profile`
			}

			intPriority, strconvErr := strconv.Atoi(priority)
			if strconvErr != nil {
				debug.Log(
					"Error: falling back to regular error handling logic due to strconv.Atoi error: %s",
					strconvErr)
				// fallthrough to the regular error handling logic
			} else {
				// to give higher priority, we need to assign a lower priority number
				args.Priority = strconv.Itoa(intPriority - 1)
				debug.Log("Re-trying nix profile install with priority %s for package %s\n",
					args.Priority,
					args.Package,
				)
				return profileInstall(args)
			}
		}

		return errors.Wrapf(err, "Command: %s", cmd)
	}
	return nil
}

func ProfileRemove(profilePath, nixpkgsCommit, pkg string) error {
	info, found := flakesPkgInfo(nixpkgsCommit, pkg)
	if !found {
		return ErrPackageNotFound
	}
	cmd := exec.Command("nix", "profile", "remove",
		"--profile", profilePath,
		"--impure", // for NIXPKGS_ALLOW_UNFREE
		info.attributeKey,
	)
	cmd.Env = AllowUnfreeEnv()
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)
	out, err := cmd.CombinedOutput()
	if bytes.Contains(out, []byte("does not match any packages")) {
		return ErrPackageNotInstalled
	}

	return errors.Wrap(err, string(out))
}

func AllowUnfreeEnv() []string {
	return append(os.Environ(), "NIXPKGS_ALLOW_UNFREE=1")
}
