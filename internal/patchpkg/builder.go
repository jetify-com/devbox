// patchpkg patches packages to fix common linker errors.
package patchpkg

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	_ "embed"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed glibc-patch.bash
var glibcPatchScript []byte

// DerivationBuilder patches an existing package.
type DerivationBuilder struct {
	// Out is the output directory that will contain the built derivation.
	// If empty it defaults to $out, which is typically set by Nix.
	Out string

	// Glibc is an optional store path to an alternative glibc version. If
	// it's set, the builder will patch ELF binaries to use its shared
	// libraries and dynamic linker.
	Glibc string

	// Gcc is an optional store path to an alternative gcc version. If
	// it's set, the builder will patch ELF binaries to use its shared
	// libraries (such as libstdc++.so).
	Gcc string

	glibcPatcher *libPatcher

	RestoreRefs bool
	bytePatches map[string][]fileSlice

	// src contains the source files of the derivation. For flakes, this is
	// anything in the flake.nix directory.
	src *packageFS
}

// NewDerivationBuilder initializes a new DerivationBuilder from the current
// Nix build environment.
func NewDerivationBuilder() (*DerivationBuilder, error) {
	d := &DerivationBuilder{}
	if err := d.init(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *DerivationBuilder) init() error {
	if d.Out == "" {
		d.Out = os.Getenv("out")
		if d.Out == "" {
			return fmt.Errorf("patchpkg: $out is empty (is this being run from a nix build?)")
		}
	}
	if d.Glibc != "" {
		if d.glibcPatcher == nil {
			d.glibcPatcher = &libPatcher{}
		}
		err := d.glibcPatcher.setGlibc(newPackageFS(d.Glibc))
		if err != nil {
			return fmt.Errorf("patchpkg: can't patch glibc using %s: %v", d.Glibc, err)
		}
	}
	if d.Gcc != "" {
		if d.glibcPatcher == nil {
			d.glibcPatcher = &libPatcher{}
		}
		err := d.glibcPatcher.setGcc(newPackageFS(d.Gcc))
		if err != nil {
			return fmt.Errorf("patchpkg: can't patch gcc using %s: %v", d.Gcc, err)
		}
	}
	if src := os.Getenv("src"); src != "" {
		d.src = newPackageFS(src)
	}
	return nil
}

// Build applies patches to a package store path and puts the result in the
// d.Out directory.
func (d *DerivationBuilder) Build(ctx context.Context, pkgStorePath string) error {
	if err := d.init(); err != nil {
		return err
	}

	slog.DebugContext(ctx, "starting build to patch package",
		"pkg", pkgStorePath, "glibc", d.Glibc, "out", d.Out)
	return d.build(ctx, newPackageFS(pkgStorePath), newPackageFS(d.Out))
}

func (d *DerivationBuilder) build(ctx context.Context, pkg, out *packageFS) error {
	// Create the derivation's $out directory.
	if err := d.copyDir(out, "."); err != nil {
		return err
	}

	if d.RestoreRefs {
		if err := d.restoreMissingRefs(ctx, pkg); err != nil {
			// Don't break the flake build if we're unable to
			// restore some of the refs. Having some is still an
			// improvement.
			slog.ErrorContext(ctx, "unable to restore all removed refs", "err", err)
		}
	}
	if err := d.findCUDA(ctx, out); err != nil {
		slog.ErrorContext(ctx, "unable to patch CUDA libraries", "err", err)
	}

	var err error
	for path, entry := range allFiles(pkg, ".") {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if path == "." {
			// Skip the $out directory - we already created it.
			continue
		}

		switch {
		case entry.IsDir():
			err = d.copyDir(out, path)
		case isSymlink(entry.Type()):
			err = d.copySymlink(pkg, out, path)
		default:
			err = d.copyFile(ctx, pkg, out, path)
		}

		if err != nil {
			return err
		}
	}

	cmd := exec.CommandContext(ctx, lookPath("bash"), "-s")
	cmd.Stdin = bytes.NewReader(glibcPatchScript)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (d *DerivationBuilder) restoreMissingRefs(ctx context.Context, pkg *packageFS) error {
	// Find store path references to build inputs that were removed
	// from Python.
	refs, err := d.findRemovedRefs(ctx, pkg)
	if err != nil {
		return err
	}

	// Group the references we want to restore by file path.
	d.bytePatches = make(map[string][]fileSlice, len(refs))
	for _, ref := range refs {
		d.bytePatches[ref.path] = append(d.bytePatches[ref.path], ref)
	}

	// If any of those references have shared libraries, add them
	// back to Python's RPATH.
	if d.glibcPatcher != nil {
		nixStore := cmp.Or(os.Getenv("NIX_STORE"), "/nix/store")
		seen := make(map[string]bool)
		for _, ref := range refs {
			storePath := filepath.Join(nixStore, string(ref.data))
			if seen[storePath] {
				continue
			}
			seen[storePath] = true
			d.glibcPatcher.prependRPATH(newPackageFS(storePath))
		}
	}
	return nil
}

func (d *DerivationBuilder) copyDir(out *packageFS, path string) error {
	path, err := out.OSPath(path)
	if err != nil {
		return err
	}
	return os.MkdirAll(path, 0o777)
}

func (d *DerivationBuilder) copyFile(ctx context.Context, pkg, out *packageFS, path string) error {
	srcFile, err := pkg.Open(path)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	src := bufio.NewReader(srcFile)
	if d.needsGlibcPatch(src, path) {
		srcPath, err := pkg.OSPath(path)
		if err != nil {
			return err
		}
		dstPath, err := out.OSPath(path)
		if err != nil {
			return err
		}
		// No need to copy the file, patchelf will do it for us.
		return d.glibcPatcher.patch(ctx, srcPath, dstPath)
	}

	stat, err := srcFile.Stat()
	if err != nil {
		return err
	}

	// We only need to copy the executable permissions of a file.
	// Nix ends up making everything in the store read-only after
	// the build is done.
	perm := fs.FileMode(0o666)
	if isExecutable(stat.Mode()) {
		perm = fs.FileMode(0o777)
	}

	dstPath, err := out.OSPath(path)
	if err != nil {
		return err
	}
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}

	for _, patch := range d.bytePatches[path] {
		_, err := dst.WriteAt(patch.data, patch.offset)
		if err != nil {
			return err
		}
	}
	return dst.Close()
}

func (d *DerivationBuilder) copySymlink(pkg, out *packageFS, path string) error {
	link, err := out.OSPath(path)
	if err != nil {
		return err
	}
	target, err := pkg.Readlink(path)
	if err != nil {
		return err
	}
	return os.Symlink(target, link)
}

func (d *DerivationBuilder) needsGlibcPatch(file *bufio.Reader, filePath string) bool {
	if d.Glibc == "" || d.glibcPatcher == nil {
		return false
	}
	if path.Dir(filePath) != "bin" {
		return false
	}

	// ELF binaries are identifiable by the first 4 magic bytes:
	// 0x7F E L F
	magic, err := file.Peek(4)
	if err != nil {
		return false
	}
	return magic[0] == 0x7F && magic[1] == 'E' && magic[2] == 'L' && magic[3] == 'F'
}

func (d *DerivationBuilder) findRemovedRefs(ctx context.Context, pkg *packageFS) ([]fileSlice, error) {
	var refs []fileSlice
	matches, err := fs.Glob(pkg, "lib/python*/_sysconfigdata_*.py")
	if err != nil {
		return nil, err
	}
	for _, name := range matches {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		matches, err := searchFile(pkg, name, reRemovedRefs)
		if err != nil {
			return nil, err
		}
		refs = append(refs, matches...)
	}

	pkgNameToHash := make(map[string]string, len(refs))
	for _, ref := range refs {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		name := string(ref.data[33:])
		if hash, ok := pkgNameToHash[name]; ok {
			copy(ref.data, hash)
			continue
		}

		re, err := regexp.Compile(`[0123456789abcdfghijklmnpqrsvwxyz]{32}-` + regexp.QuoteMeta(name) + `([$"'{}/[\] \t\r\n]|$)`)
		if err != nil {
			return nil, err
		}
		match := searchEnv(re)
		if match == "" {
			return nil, fmt.Errorf("can't find hash to restore store path reference %q in %q: regexp %q returned 0 matches", ref.data, ref.path, re)
		}
		hash := match[:32]
		pkgNameToHash[name] = hash
		copy(ref.data, hash)
		slog.DebugContext(ctx, "restored store ref", "ref", ref)
	}
	return refs, nil
}

func (d *DerivationBuilder) findCUDA(ctx context.Context, out *packageFS) error {
	if d.src == nil {
		return fmt.Errorf("patch flake didn't set $src to the path to its source tree")
	}

	pattern := "lib/libcuda.so*"
	slog.DebugContext(ctx, "looking for system CUDA libraries in flake", "glob", filepath.Join(d.src.storePath, "lib/libcuda.so*"))
	glob, err := fs.Glob(d.src, pattern)
	if err != nil {
		return fmt.Errorf("glob system libraries: %v", err)
	}
	if len(glob) == 0 {
		slog.DebugContext(ctx, "no system CUDA libraries found in flake")
	} else {
		err := d.copyDir(out, "lib")
		if err != nil {
			return fmt.Errorf("copy system library: %v", err)
		}
	}
	for _, lib := range glob {
		slog.DebugContext(ctx, "found system CUDA library in flake", "path", lib)

		err := d.copyFile(ctx, d.src, out, lib)
		if err != nil {
			return fmt.Errorf("copy system library: %v", err)
		}
		need, err := out.OSPath(lib)
		if err != nil {
			return fmt.Errorf("get absolute path to library: %v", err)
		}
		d.glibcPatcher.needed = append(d.glibcPatcher.needed, need)

		slog.DebugContext(ctx, "added DT_NEEDED entry for system CUDA library", "path", need)
	}

	slog.DebugContext(ctx, "looking for nix libraries in $patchDependencies")
	deps := os.Getenv("patchDependencies")
	if strings.TrimSpace(deps) == "" {
		slog.DebugContext(ctx, "$patchDependencies is empty")
		return nil
	}
	for _, pkg := range strings.Split(deps, " ") {
		slog.DebugContext(ctx, "checking for nix libraries in package", "pkg", pkg)

		pkgFS := newPackageFS(pkg)
		libs, err := fs.Glob(pkgFS, "lib*/*.so*")
		if err != nil {
			return fmt.Errorf("glob nix package libraries: %v", err)
		}

		sonameRegexp := regexp.MustCompile(`(^|/).+\.so\.\d+`)
		for _, lib := range libs {
			if !sonameRegexp.MatchString(lib) {
				continue
			}
			need, err := pkgFS.OSPath(lib)
			if err != nil {
				return fmt.Errorf("get absolute path to nix package library: %v", err)
			}
			d.glibcPatcher.needed = append(d.glibcPatcher.needed, need)

			slog.DebugContext(ctx, "added DT_NEEDED entry for nix library", "path", need)
		}
	}
	return nil
}

// packageFS is the tree of files for a package in the Nix store.
type packageFS struct {
	fs.FS
	storePath string
}

// newPackageFS returns a packageFS for the given store path.
func newPackageFS(storePath string) *packageFS {
	return &packageFS{
		FS:        os.DirFS(storePath),
		storePath: storePath,
	}
}

// Readlink returns the destination of a symlink.
func (p *packageFS) Readlink(path string) (string, error) {
	osPath, err := p.OSPath(path)
	if err != nil {
		return "", err
	}
	// TODO(gcurtis): check that the symlink isn't absolute or points
	// outside the Nix store.
	return os.Readlink(osPath)
}

// OSPath translates a package-relative path to an operating system path.
func (p *packageFS) OSPath(path string) (string, error) {
	local, err := filepath.Localize(path)
	if err != nil {
		return "", err
	}
	return filepath.Join(p.storePath, local), nil
}

// allFiles iterates over all files in fsys starting at root. It silently
// ignores errors.
func allFiles(fsys fs.FS, root string) iter.Seq2[string, fs.DirEntry] {
	return func(yield func(string, fs.DirEntry) bool) {
		_ = fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
			if err == nil {
				if !yield(path, d) {
					return filepath.SkipAll
				}
			}
			return nil
		})
	}
}

// lookPath is like [exec.lookPath], but first checks if there's an environment
// variable with the name prog. If there is, it returns $prog/bin/prog instead
// of consulting PATH.
//
// For example, lookPath would be able to find bash and patchelf in the
// following derivation:
//
//	derivation {
//	  inherit (nixpkgs.legacyPackages.x86_64-linux) bash patchelf;
//	  builder = devbox;
//	}
func lookPath(prog string) string {
	pkgPath := os.Getenv(prog)
	if pkgPath == "" {
		return prog
	}
	return filepath.Join(pkgPath, "bin", prog)
}

func isExecutable(mode fs.FileMode) bool { return mode&0o111 != 0 }
func isSymlink(mode fs.FileMode) bool    { return mode&fs.ModeSymlink != 0 }
