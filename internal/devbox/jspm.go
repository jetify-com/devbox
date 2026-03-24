package devbox

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/samber/lo"
	"go.jetify.com/devbox/internal/devpkg"
	"go.jetify.com/devbox/internal/devpkg/pkgtype"
	"go.jetify.com/devbox/internal/nix"
	"go.jetify.com/devbox/internal/ux"
)

// InstallJSPMPackages installs JS packages via their package managers.
// Called from installPackages(), parallel to InstallRunXPackages().
func (d *Devbox) InstallJSPMPackages(ctx context.Context) error {
	jspmPkgs := lo.Filter(d.InstallablePackages(), devpkg.IsJSPM)
	if len(jspmPkgs) == 0 {
		return nil
	}

	for _, pkg := range jspmPkgs {
		mgr := pkg.JSPMType()
		name, version := pkg.JSPMPackageName()
		if version == "" {
			version = "latest"
		}

		// Check version sync with package.json
		d.syncJSPMVersion(mgr, name, version)

		// Install the package
		pkgSpec := name
		if version != "" {
			pkgSpec = name + "@" + version
		}
		ux.Finfof(d.stderr, "Installing %s via %s\n", pkgSpec, mgr)

		if err := d.jspmRunCommand(ctx, string(mgr), "add", pkgSpec); err != nil {
			return fmt.Errorf("error installing %s package %s: %w", mgr, name, err)
		}
	}
	return nil
}

// RemoveJSPMPackages removes JS packages via their package managers.
func (d *Devbox) RemoveJSPMPackages(ctx context.Context, pkgs []string) error {
	for _, raw := range pkgs {
		if !pkgtype.IsJSPM(raw) {
			continue
		}
		mgr := pkgtype.JSPMType(raw)
		name, _ := pkgtype.JSPMPackageName(raw)

		ux.Finfof(d.stderr, "Removing %s via %s\n", name, mgr)

		if err := d.jspmRunCommand(ctx, string(mgr), "remove", name); err != nil {
			ux.Fwarningf(d.stderr, "warning: failed to remove %s via %s: %s\n", name, mgr, err)
		}
	}
	return nil
}

// UpdateJSPMPackage updates a JS package via its package manager.
func (d *Devbox) UpdateJSPMPackage(ctx context.Context, pkg *devpkg.Package) error {
	mgr := pkg.JSPMType()
	name, version := pkg.JSPMPackageName()

	if version != "" && version != "latest" {
		// Specific version: use add to pin
		pkgSpec := name + "@" + version
		ux.Finfof(d.stderr, "Updating %s to %s via %s\n", name, version, mgr)
		return d.jspmRunCommand(ctx, string(mgr), "add", pkgSpec)
	}

	// For latest or unversioned, use update
	var updateCmd string
	switch mgr {
	case pkgtype.Yarn:
		updateCmd = "upgrade"
	default:
		updateCmd = "update"
	}

	ux.Finfof(d.stderr, "Updating %s via %s\n", name, mgr)
	return d.jspmRunCommand(ctx, string(mgr), updateCmd, name)
}

// JSPMPaths creates symlinks for JSPM package binaries and returns the bin paths.
// Called from computeEnv(), parallel to RunXPaths().
func (d *Devbox) JSPMPaths(ctx context.Context) (string, error) {
	jspmPkgs := lo.Filter(d.InstallablePackages(), devpkg.IsJSPM)
	if len(jspmPkgs) == 0 {
		return "", nil
	}

	// Collect unique managers in use
	managers := map[pkgtype.JSPackageManager]bool{}
	for _, pkg := range jspmPkgs {
		managers[pkg.JSPMType()] = true
	}

	var binPaths []string
	for mgr := range managers {
		binPath := jspmBinPath(d.projectDir, mgr)
		if err := os.RemoveAll(binPath); err != nil {
			return "", err
		}
		if err := os.MkdirAll(binPath, 0o755); err != nil {
			return "", err
		}

		// Symlink binaries from node_modules/.bin/ to our virtenv bin dir
		nodeModulesBin := filepath.Join(d.projectDir, "node_modules", ".bin")
		if entries, err := os.ReadDir(nodeModulesBin); err == nil {
			for _, entry := range entries {
				src := filepath.Join(nodeModulesBin, entry.Name())
				dst := filepath.Join(binPath, entry.Name())
				if err := os.Symlink(src, dst); err != nil && !os.IsExist(err) {
					return "", err
				}
			}
		}

		binPaths = append(binPaths, binPath)
	}

	return strings.Join(binPaths, string(filepath.ListSeparator)), nil
}

// jspmRunCommand runs a JS package manager command. It looks for the binary
// in the devbox nix profile first (which is already installed by the time
// JSPM packages are installed), then falls back to the system PATH.
func (d *Devbox) jspmRunCommand(ctx context.Context, manager string, args ...string) error {
	// Build a PATH that includes the nix profile bin directory.
	// By the time JSPM packages are installed, nix packages (including nodejs/pnpm)
	// are already in the nix profile.
	profileBin := nix.ProfileBinPath(d.projectDir)
	path := profileBin + string(filepath.ListSeparator) + os.Getenv("PATH")

	// Look up the manager binary in our augmented PATH
	managerPath, err := lookPathIn(manager, path)
	if err != nil {
		return fmt.Errorf(
			"%s not found. Add nodejs or %s to your devbox.json packages",
			manager, manager,
		)
	}

	cmd := exec.CommandContext(ctx, managerPath, args...)
	cmd.Dir = d.projectDir
	cmd.Env = append(os.Environ(), "PATH="+path)
	cmd.Stdout = d.stderr // use stderr for install output
	cmd.Stderr = d.stderr
	return cmd.Run()
}

// lookPathIn searches for an executable in the given PATH string.
func lookPathIn(file, pathEnv string) (string, error) {
	for _, dir := range filepath.SplitList(pathEnv) {
		path := filepath.Join(dir, file)
		if fi, err := os.Stat(path); err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0 {
			return path, nil
		}
	}
	return "", fmt.Errorf("%s not found in PATH", file)
}

// syncJSPMVersion checks if the package.json version matches devbox version and warns if not.
func (d *Devbox) syncJSPMVersion(mgr pkgtype.JSPackageManager, name, devboxVersion string) {
	pkgJSONPath := filepath.Join(d.projectDir, "package.json")
	data, err := os.ReadFile(pkgJSONPath)
	if err != nil {
		// No package.json yet; the package manager will create one.
		return
	}

	var pkgJSON map[string]any
	if err := json.Unmarshal(data, &pkgJSON); err != nil {
		return
	}

	// Check both dependencies and devDependencies
	for _, depKey := range []string{"dependencies", "devDependencies"} {
		deps, ok := pkgJSON[depKey].(map[string]any)
		if !ok {
			continue
		}
		existingVersion, ok := deps[name].(string)
		if !ok {
			continue
		}

		// Compare versions. Strip leading ^ ~ >= etc for comparison.
		cleanExisting := strings.TrimLeft(existingVersion, "^~>=<")
		if devboxVersion != "latest" && cleanExisting != devboxVersion {
			ux.Fwarningf(
				d.stderr,
				"devbox and package.json version of %s don't match (devbox: %s, package.json: %s). "+
					"Run \"devbox add %s:%s@%s\" to fix.\n",
				name, devboxVersion, existingVersion,
				mgr, name, cleanExisting,
			)
		}
		return
	}
}

func jspmBinPath(projectDir string, mgr pkgtype.JSPackageManager) string {
	return filepath.Join(projectDir, ".devbox", "virtenv", string(mgr), "bin")
}
