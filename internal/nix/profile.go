package nix

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"go.jetpack.io/devbox/internal/redact"
)

const DefaultPriority = 5

type NixProfileList []*NixProfileListItem

// ProfileListItems returns a list of the installed packages
func ProfileListItems(writer io.Writer, profileDir string) (NixProfileList, error) {
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
		return nil, redact.Errorf("error creating stdout pipe: %w", redact.Safe(err))
	}
	if err := cmd.Start(); err != nil {
		return nil, redact.Errorf("error starting \"nix profile list\" command: %w", err)
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
		return items, redact.Errorf("error running \"nix profile list\": %w", err)
	}
	return items, nil
}

type ProfileListIndexArgs struct {
	// For performance you can reuse the same list in multiple operations if you
	// are confident index has not changed.
	List       NixProfileList
	Writer     io.Writer
	Pkg        string
	ProjectDir string
	ProfileDir string
}

func ProfileListIndex(args *ProfileListIndexArgs) (int, error) {
	var err error
	list := args.List
	if list == nil {
		list, err = ProfileListItems(args.Writer, args.ProfileDir)
		if err != nil {
			return -1, err
		}
	}

	input := InputFromString(args.Pkg, args.ProjectDir)

	for _, item := range list {
		name, err := item.packageName()
		if err != nil {
			return -1, err
		}

		existing := InputFromString(name, args.ProjectDir)

		if input.equals(existing) {
			return item.index, nil
		}
	}
	return -1, ErrPackageNotFound
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
		return nil, redact.Errorf("error parsing \"nix profile list\" output: line is missing index: %s", line)
	}
	index, err := strconv.Atoi(scanner.Text())
	if err != nil {
		return nil, redact.Errorf("error parsing \"nix profile list\" output: %w: %s", line)
	}

	if !scanner.Scan() {
		return nil, redact.Errorf("error parsing \"nix profile list\" output: line is missing unlockedReference: %s", line)
	}
	unlockedReference := scanner.Text()

	if !scanner.Scan() {
		return nil, redact.Errorf("error parsing \"nix profile list\" output: line is missing lockedReference: %s", line)
	}
	lockedReference := scanner.Text()

	if !scanner.Scan() {
		return nil, redact.Errorf("error parsing \"nix profile list\" output: line is missing nixStorePath: %s", line)
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
		return "", redact.Errorf(
			"expected to find # in lockedReference: %s from NixProfileListItem: %s",
			redact.Safe(item.lockedReference),
			item,
		)
	}
	return attrPath, nil
}

// packageName parses the package name from the NixProfileListItem.lockedReference
//
// For example:
// if NixProfileListItem.lockedReference = github:NixOS/nixpkgs/52e3e80afff4b16ccb7c52e9f0f5220552f03d04#legacyPackages.x86_64-darwin.go_1_19
// then AttributePath = legacyPackages.x86_64-darwin.go_1_19
// and then packageName = go_1_19
func (item *NixProfileListItem) packageName() (string, error) {
	// Since unlocked references should already have absolute paths, projectDir is not needed
	input := InputFromString(item.unlockedReference, "" /*projectDir*/)
	if input.IsFlake() {
		// TODO(landau): this needs to be normalized so that we can compare
		// packages.aarch64-darwin.hello and hello and determine that they are
		// the same package.
		return item.unlockedReference, nil
	}

	// TODO(landau/savil): Should we use unlocked reference instead?
	// I'm not sure we gain anything by using the locked reference. since
	// the user specifies only the package name when installing
	attrPath, err := item.AttributePath()
	if err != nil {
		return "", err
	}

	parts := strings.Split(attrPath, ".")
	if len(parts) < 2 {
		return "", redact.Errorf(
			"expected >= 2 parts for AttributePath in \"nix profile list\" output, but got %d parts: %s",
			redact.Safe(len(parts)),
			redact.Safe(attrPath),
		)
	}

	return strings.Join(parts[2:], "."), nil
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
	ExtraFlags        []string
	NixpkgsCommit     string
	Package           string
	ProfilePath       string
	ProjectDir        string
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

	cmd := exec.Command(
		"nix", "profile", "install",
		"--profile", args.ProfilePath,
		"--impure", // for NIXPKGS_ALLOW_UNFREE
		// Using an arbitrary priority to avoid conflicts with other packages.
		// Note that this is not really the priority we care about, since we
		// use the flake.nix to specify the priority.
		"--priority", nextPriority(args.ProfilePath),
		flakePath(args),
	)
	cmd.Env = AllowUnfreeEnv()
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)
	cmd.Args = append(cmd.Args, args.ExtraFlags...)

	// If nix profile install runs as tty, the output is much nicer. If we ever
	// need to change this to our own writers, consider that you may need
	// to implement your own nicer output. --print-build-logs flag may be useful.
	cmd.Stdin = os.Stdin
	cmd.Stdout = args.Writer
	cmd.Stderr = args.Writer

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(args.Writer, "%s: ", stepMsg)
		color.New(color.FgRed).Fprintf(args.Writer, "Fail\n")
		return redact.Errorf("error running \"nix profile install\": %w", err)
	}

	fmt.Fprintf(args.Writer, "%s: ", stepMsg)
	color.New(color.FgGreen).Fprintf(args.Writer, "Success\n")
	return nil
}

func ProfileRemove(profilePath, nixpkgsCommit, pkg string) error {
	info, found := PkgInfo(nixpkgsCommit, pkg)
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
	if err != nil {
		return redact.Errorf("error running \"nix profile remove\": %s: %w", out, err)
	}
	return nil
}

func AllowUnfreeEnv() []string {
	return append(os.Environ(), "NIXPKGS_ALLOW_UNFREE=1")
}

type manifest struct {
	Elements []struct {
		Priority int `json:"priority"`
	} `json:"elements"`
}

func readManifest(profilePath string) (manifest, error) {
	data, err := os.ReadFile(filepath.Join(profilePath, "manifest.json"))
	if os.IsNotExist(err) {
		return manifest{}, nil
	} else if err != nil {
		return manifest{}, err
	}

	var m manifest
	return m, json.Unmarshal(data, &m)
}

func nextPriority(profilePath string) string {
	// error is ignored because it's ok if the file doesn't exist
	m, _ := readManifest(profilePath)
	max := DefaultPriority
	for _, e := range m.Elements {
		if e.Priority > max {
			max = e.Priority
		}
	}
	// Each subsequent package gets a lower priority. This matches how flake.nix
	// behaves
	return fmt.Sprintf("%d", max+1)
}

func flakePath(args *ProfileInstallArgs) string {
	input := InputFromString(args.Package, args.ProjectDir)
	if input.IsFlake() {
		return input.String()
	}

	return FlakeNixpkgs(args.NixpkgsCommit) + "#" + args.Package
}
