package shellgen

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/trace"
	"slices"
	"strings"

	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/patchpkg"
	"go.jetpack.io/devbox/nix/flake"
)

// flakePlan contains the data to populate the top level flake.nix file
// that builds the devbox environment
type flakePlan struct {
	NixpkgsInfo *NixpkgsInfo
	Packages    []*devpkg.Package
	FlakeInputs []flakeInput
	System      string
}

func newFlakePlan(ctx context.Context, devbox devboxer) (*flakePlan, error) {
	ctx, task := trace.NewTask(ctx, "devboxFlakePlan")
	defer task.End()

	for _, pluginConfig := range devbox.Config().IncludedPluginConfigs() {
		if err := devbox.PluginManager().CreateFilesForConfig(pluginConfig); err != nil {
			return nil, err
		}
	}

	packages := devbox.InstallablePackages()

	// Fill the NarInfo Cache concurrently as a perf-optimization, prior to invoking
	// IsInBinaryCache in flakeInputs() and in the flake.nix template.
	if err := devpkg.FillNarInfoCache(ctx, packages...); err != nil {
		return nil, err
	}

	flakeInputs := flakeInputs(ctx, packages)
	nixpkgsInfo := getNixpkgsInfo(devbox.Config().NixPkgsCommitHash())

	// This is an optimization. Try to reuse the nixpkgs info from the flake
	// inputs to avoid introducing a new one.
	for _, input := range flakeInputs {
		if input.IsNixpkgs() {
			nixpkgsInfo = getNixpkgsInfo(input.HashFromNixPkgsURL())
			break
		}
	}

	return &flakePlan{
		FlakeInputs: flakeInputs,
		NixpkgsInfo: nixpkgsInfo,
		Packages:    packages,
		System:      nix.System(),
	}, nil
}

func (f *flakePlan) needsGlibcPatch() bool {
	for _, in := range f.FlakeInputs {
		if in.URL == glibcPatchFlakeRef {
			return true
		}
	}
	return false
}

type glibcPatchFlake struct {
	// DevboxExecutable is the absolute path to the Devbox binary to use as
	// the flake's builder. It must not be the wrapper script.
	DevboxExecutable string

	// NixpkgsGlibcFlakeRef is a flake reference to the nixpkgs flake
	// containing the new glibc package.
	NixpkgsGlibcFlakeRef string

	// Inputs is the attribute set of flake inputs. The key is the input
	// name and the value is a flake reference.
	Inputs map[string]string

	// Outputs is the attribute set of flake outputs. It follows the
	// standard flake output schema of system.name = derivation. The
	// derivation can be any valid Nix expression.
	Outputs struct {
		Packages map[string]map[string]string
	}

	// Dependencies is set of extra packages that are dependencies of the
	// patched packages. For example, a patched Python interpreter might
	// need CUDA packages, but the CUDA packages themselves don't need
	// patching.
	Dependencies []string
}

func newGlibcPatchFlake(nixpkgsGlibcRev string, packages []*devpkg.Package) (glibcPatchFlake, error) {
	// Get the path to the actual devbox binary (not the /usr/bin/devbox
	// wrapper) so the flake build can use it.
	exe, err := os.Executable()
	if err != nil {
		return glibcPatchFlake{}, err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return glibcPatchFlake{}, err
	}

	flake := glibcPatchFlake{
		DevboxExecutable:     exe,
		NixpkgsGlibcFlakeRef: "flake:nixpkgs/" + nixpkgsGlibcRev,
	}
	for _, pkg := range packages {
		// Check to see if this is a CUDA package. If so, we need to add
		// it to the flake dependencies so that we can patch other
		// packages to reference it (like Python).
		relAttrPath, err := flake.systemRelativeAttrPath(pkg)
		if err != nil {
			return glibcPatchFlake{}, err
		}
		if strings.HasPrefix(relAttrPath, "cudaPackages") {
			if err := flake.addDependency(pkg); err != nil {
				return glibcPatchFlake{}, err
			}
		}

		if !pkg.Patch {
			continue
		}
		if err := flake.addOutput(pkg); err != nil {
			return glibcPatchFlake{}, err
		}
	}
	return flake, nil
}

// addInput adds a flake input that provides pkg.
func (g *glibcPatchFlake) addInput(pkg *devpkg.Package) error {
	if g.Inputs == nil {
		g.Inputs = make(map[string]string)
	}
	installable, err := pkg.FlakeInstallable()
	if err != nil {
		return err
	}
	inputName := pkg.FlakeInputName()
	g.Inputs[inputName] = installable.Ref.String()
	return nil
}

// addOutput adds a flake output that provides the patched version of pkg.
func (g *glibcPatchFlake) addOutput(pkg *devpkg.Package) error {
	if err := g.addInput(pkg); err != nil {
		return err
	}

	relAttrPath, err := g.systemRelativeAttrPath(pkg)
	if err != nil {
		return err
	}
	if g.Outputs.Packages == nil {
		g.Outputs.Packages = map[string]map[string]string{nix.System(): {}}
	}
	if cached, err := pkg.IsInBinaryCache(); err == nil && cached {
		if expr, err := g.fetchClosureExpr(pkg); err == nil {
			g.Outputs.Packages[nix.System()][relAttrPath] = expr
			return nil
		}
	}

	inputAttrPath, err := g.inputRelativeAttrPath(pkg)
	if err != nil {
		return err
	}
	g.Outputs.Packages[nix.System()][relAttrPath] = inputAttrPath
	return nil
}

// addDependency adds pkg to the derivation's patchDependencies attribute,
// making it available at patch build-time.
func (g *glibcPatchFlake) addDependency(pkg *devpkg.Package) error {
	if err := g.addInput(pkg); err != nil {
		return err
	}
	inputAttrPath, err := g.inputRelativeAttrPath(pkg)
	if err != nil {
		return err
	}

	installable, err := pkg.FlakeInstallable()
	if err != nil {
		return err
	}
	switch installable.Outputs {
	case flake.DefaultOutputs:
		expr := "selectDefaultOutputs " + inputAttrPath
		g.Dependencies = append(g.Dependencies, expr)
	case flake.AllOutputs:
		expr := "selectAllOutputs " + inputAttrPath
		g.Dependencies = append(g.Dependencies, expr)
	default:
		expr := fmt.Sprintf("selectOutputs %s %q", inputAttrPath, installable.SplitOutputs())
		g.Dependencies = append(g.Dependencies, expr)
	}
	return nil
}

// systemRelativeAttrPath strips any leading "legacyPackages.<system>" prefix
// from a package's attribute path.
func (g *glibcPatchFlake) systemRelativeAttrPath(pkg *devpkg.Package) (string, error) {
	installable, err := pkg.FlakeInstallable()
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(installable.AttrPath, "legacyPackages") {
		// Remove the legacyPackages.<system> prefix.
		return strings.SplitN(installable.AttrPath, ".", 3)[2], nil
	}
	return installable.AttrPath, nil
}

// inputRelativeAttrPath joins the package's corresponding flake input with its
// attribute path.
func (g *glibcPatchFlake) inputRelativeAttrPath(pkg *devpkg.Package) (string, error) {
	relAttrPath, err := g.systemRelativeAttrPath(pkg)
	if err != nil {
		return "", err
	}
	atrrPath := strings.Join([]string{"pkgs", pkg.FlakeInputName(), nix.System(), relAttrPath}, ".")
	return atrrPath, nil
}

// TODO: this only handles the first store path, but we should handle all of them
func (g *glibcPatchFlake) fetchClosureExpr(pkg *devpkg.Package) (string, error) {
	storePaths, err := pkg.InputAddressedPaths()
	if err != nil {
		return "", err
	}
	if len(storePaths) == 0 {
		return "", fmt.Errorf("no store path for package %s", pkg.Raw)
	}
	return fmt.Sprintf(`builtins.fetchClosure {
  fromStore = "%s";
  fromPath = "%s";
  inputAddressed = true;
}`, "devpkg.BinaryCache", storePaths[0]), nil
}

// copySystemCUDALib searches for the system's libcuda.so shared library and
// copies it in the flake's lib directory.
func (g *glibcPatchFlake) copySystemCUDALib(flakeDir string) error {
	slog.Debug("found CUDA package in devbox environment, attempting to find system driver libraries")

	searchPath := slices.Concat(
		patchpkg.EnvLDLibrarySearchPath,
		patchpkg.EnvLibrarySearchPath,
		patchpkg.SystemLibSearchPaths,
		patchpkg.CUDALibSearchPaths,
	)
	for lib := range patchpkg.FindSharedLibrary("libcuda.so", searchPath...) {
		logger := slog.With("lib", lib)
		logger.Debug("found potential system CUDA library")

		stat, err := lib.Stat()
		if err != nil {
			logger.Error("skipping system CUDA library because of stat error", "err", err)
		}
		const mib = 1 << 20
		if stat.Size() < 1*mib {
			logger.Debug("skipping system CUDA library because it looks like a stub (size < 1 MiB)", "size", stat.Size())
			continue
		}
		if lib.Soname == "" {
			logger.Debug("skipping system CUDA library because it's missing a soname")
			continue
		}

		libDir := filepath.Join(flakeDir, "lib")
		if err := lib.CopyAndLink(libDir); err == nil {
			slog.Debug("copied system CUDA library to flake directory", "dst", libDir)
		} else {
			slog.Error("can't copy system CUDA library to flake directory", "err", err)
		}
		return err
	}
	return fmt.Errorf("can't find the system CUDA library")
}

func (g *glibcPatchFlake) writeTo(dir string) error {
	wantCUDA := slices.ContainsFunc(g.Dependencies, func(dep string) bool {
		return strings.Contains(dep, "cudaPackages")
	})
	if wantCUDA {
		err := g.copySystemCUDALib(dir)
		if err != nil {
			slog.Debug("error copying system libcuda.so to flake", "dir", dir)
		}
	}
	return writeFromTemplate(dir, g, "glibc-patch.nix", "flake.nix")
}
