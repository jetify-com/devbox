// patchpkg patches packages to fix common linker errors.
package patchpkg

import (
	"bufio"
	"bytes"
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
	Glibc        string
	glibcPatcher glibcPatcher
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
		var err error
		d.glibcPatcher, err = newGlibcPatcher(newPackageFS(d.Glibc))
		if err != nil {
			return fmt.Errorf("patchpkg: can't patch glibc using %s: %v", d.Glibc, err)
		}
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
	var err error
	for path, entry := range allFiles(pkg, ".") {
		if ctx.Err() != nil {
			return err
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

func (d *DerivationBuilder) copyDir(out *packageFS, path string) error {
	path, err := out.OSPath(path)
	if err != nil {
		return err
	}
	return os.Mkdir(path, 0o777)
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
	if d.Glibc == "" {
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
