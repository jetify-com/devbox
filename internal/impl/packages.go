package impl

import (
	"encoding/json"
	"fmt"
	"io/fs"
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
	"go.jetpack.io/devbox/internal/xdg"
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
			"--priority", d.getPackagePriority(pkg),
			"--impure", // Needed to allow flags from environment to be used.
			nix.FlakeNixpkgs(d.cfg.Nixpkgs.Commit)+"#"+pkg,
		)
		cmd.Args = append(cmd.Args, nix.ExperimentalFlags()...)
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

// ensureNixpkgsPrefetched runs the prefetch step to download the flake of the registry
func (d *Devbox) ensureNixpkgsPrefetched() error {
	// Look up the cached map of commitHash:nixStoreLocation
	commitToLocation, err := d.nixpkgsCommitFileContents()
	if err != nil {
		return err
	}

	// Check if this nixpkgs.Commit is located in the local /nix/store
	location, isPresent := commitToLocation[d.cfg.Nixpkgs.Commit]
	if isPresent {
		if fi, err := os.Stat(location); err == nil && fi.IsDir() {
			// The nixpkgs for this commit hash is present, so we don't need to prefetch
			return nil
		}
	}

	fmt.Fprintf(d.writer, "Ensuring nixpkgs registry is downloaded.\n")
	cmd := exec.Command(
		"nix", "flake", "prefetch",
		nix.FlakeNixpkgs(d.cfg.Nixpkgs.Commit),
	)
	cmd.Args = append(cmd.Args, nix.ExperimentalFlags()...)
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

	return d.saveToNixpkgsCommitFile(commitToLocation)
}

func (d *Devbox) nixpkgsCommitFileContents() (map[string]string, error) {
	path := nixpkgsCommitFilePath()
	if !fileutil.Exists(path) {
		return map[string]string{}, nil
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	commitToLocation := map[string]string{}
	if err := json.Unmarshal(contents, &commitToLocation); err != nil {
		return nil, errors.WithStack(err)
	}
	return commitToLocation, nil
}

func (d *Devbox) saveToNixpkgsCommitFile(commitToLocation map[string]string) error {
	// Make a query to get the /nix/store path for this commit hash.
	cmd := exec.Command("nix", "flake", "prefetch", "--json",
		nix.FlakeNixpkgs(d.cfg.Nixpkgs.Commit),
	)
	cmd.Args = append(cmd.Args, nix.ExperimentalFlags()...)
	out, err := cmd.Output()
	if err != nil {
		return errors.WithStack(err)
	}

	// read the json response
	var prefetchData struct {
		StorePath string `json:"storePath"`
	}
	if err := json.Unmarshal(out, &prefetchData); err != nil {
		return errors.WithStack(err)
	}

	// Ensure the nixpkgs commit file path exists so we can write an update to it
	path := nixpkgsCommitFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil && !errors.Is(err, fs.ErrExist) {
		return errors.WithStack(err)
	}

	// write to the map, jsonify it, and write that json to the nixpkgsCommit file
	commitToLocation[d.cfg.Nixpkgs.Commit] = prefetchData.StorePath
	serialized, err := json.Marshal(commitToLocation)
	if err != nil {
		return errors.WithStack(err)
	}

	err = os.WriteFile(path, serialized, 0644)
	return errors.WithStack(err)
}

func nixpkgsCommitFilePath() string {
	cacheDir := xdg.CacheSubpath("devbox")
	return filepath.Join(cacheDir, "nixpkgs.json")
}
