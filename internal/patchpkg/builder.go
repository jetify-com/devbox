// patchpkg patches packages to fix common linker errors.
package patchpkg

import (
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
	"path/filepath"
)

//go:embed glibc-patch.bash
var glibcPatchScript []byte

// DerivationBuilder patches an existing package.
type DerivationBuilder struct {
	// Out is the output directory that will contain the built derivation.
	// If empty it defaults to $out, which is typically set by Nix.
	Out string
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
	return nil
}

// Build applies patches to a package store path and puts the result in the
// d.Out directory.
func (d *DerivationBuilder) Build(ctx context.Context, pkgStorePath string) error {
	slog.DebugContext(ctx, "starting build of patched package", "pkg", pkgStorePath, "out", d.Out)

	var err error
	pkgFS := os.DirFS(pkgStorePath)
	for path, entry := range allFiles(pkgFS, ".") {
		switch {
		case entry.IsDir():
			err = d.copyDir(path)
		case isSymlink(entry.Type()):
			err = d.copySymlink(pkgStorePath, path)
		default:
			err = d.copyFile(pkgFS, path)
		}

		if err != nil {
			return err
		}
	}

	bash := filepath.Join(os.Getenv("bash"), "bin/bash")
	cmd := exec.CommandContext(ctx, bash, "-s")
	cmd.Stdin = bytes.NewReader(glibcPatchScript)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (d *DerivationBuilder) copyDir(path string) error {
	osPath, err := filepath.Localize(path)
	if err != nil {
		return err
	}
	return os.Mkdir(filepath.Join(d.Out, osPath), 0o777)
}

func (d *DerivationBuilder) copyFile(pkgFS fs.FS, path string) error {
	src, err := pkgFS.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()

	stat, err := src.Stat()
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

	osPath, err := filepath.Localize(path)
	if err != nil {
		return err
	}
	dstPath := filepath.Join(d.Out, osPath)

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

func (d *DerivationBuilder) copySymlink(pkgStorePath, path string) error {
	// The fs package doesn't support symlinks, so we need to convert the
	// path back to an OS path to see what it points to.
	osPath, err := filepath.Localize(path)
	if err != nil {
		return err
	}
	target, err := os.Readlink(filepath.Join(pkgStorePath, osPath))
	if err != nil {
		return err
	}
	// TODO(gcurtis): translate absolute symlink targets to relative paths.
	return os.Symlink(target, filepath.Join(d.Out, osPath))
}

// RegularFiles iterates over all files in fsys starting at root. It silently
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

func isExecutable(mode fs.FileMode) bool { return mode&0o111 != 0 }
func isSymlink(mode fs.FileMode) bool    { return mode&fs.ModeSymlink != 0 }
