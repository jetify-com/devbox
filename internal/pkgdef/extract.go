// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mholt/archives"
)

// extractArchive unpacks archivePath into dest, dropping the first
// stripComponents path elements of every entry. The format (tar with gz,
// xz, bz2 or zstd compression, or zip) is detected from the content.
//
// Entries are sanitized: paths that escape dest, absolute symlink targets
// and symlinks whose target escapes dest are rejected, and an existing
// symlink is never followed when writing a file.
func extractArchive(ctx context.Context, archivePath, dest string, stripComponents *int) error {
	strip := 0
	if stripComponents != nil {
		strip = *stripComponents
	}

	archive, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer archive.Close()

	format, stream, err := archives.Identify(ctx, filepath.Base(archivePath), archive)
	if err != nil {
		return fmt.Errorf("unrecognized archive format: %w", err)
	}
	extractor, ok := format.(archives.Extractor)
	if !ok {
		return fmt.Errorf("%s is not an archive", filepath.Base(archivePath))
	}

	handler := func(_ context.Context, info archives.FileInfo) error {
		rel, ok := stripPath(info.NameInArchive, strip)
		if !ok {
			return nil
		}
		abs := filepath.Join(dest, rel)
		if !isInside(dest, abs) {
			return fmt.Errorf("archive entry %q escapes the destination", info.NameInArchive)
		}

		mode := info.Mode()
		switch {
		case mode.IsDir():
			return os.MkdirAll(abs, 0o755)
		case mode&os.ModeSymlink != 0:
			return writeSymlink(dest, abs, info.LinkTarget, info.NameInArchive)
		case mode.IsRegular():
			return writeRegular(abs, info)
		default:
			// Devices, sockets and the like have no place in a package.
			return nil
		}
	}
	return extractor.Extract(ctx, stream, handler)
}

// stripPath cleans an archive path and drops its first strip components. It
// reports false when the entry should be skipped: it is one of the stripped
// directories or it has no name left.
func stripPath(name string, strip int) (string, bool) {
	cleaned := path.Clean("/" + filepath.ToSlash(name)) // "/" anchors "..", stripped below
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "" || cleaned == "." {
		return "", false
	}
	parts := strings.Split(cleaned, "/")
	if len(parts) <= strip {
		return "", false
	}
	return filepath.FromSlash(strings.Join(parts[strip:], "/")), true
}

func isInside(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func writeSymlink(dest, abs, target, name string) error {
	if target == "" || filepath.IsAbs(target) {
		return fmt.Errorf("archive entry %q: unsupported symlink target %q", name, target)
	}
	resolved := filepath.Join(filepath.Dir(abs), filepath.FromSlash(target))
	if !isInside(dest, resolved) {
		return fmt.Errorf("archive entry %q: symlink target %q escapes the destination", name, target)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	if err := os.Remove(abs); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Symlink(filepath.FromSlash(target), abs)
}

func writeRegular(abs string, info archives.FileInfo) error {
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	// Never write through a symlink that an earlier entry planted.
	if existing, err := os.Lstat(abs); err == nil && existing.Mode()&os.ModeSymlink != 0 {
		if err := os.Remove(abs); err != nil {
			return err
		}
	}

	from, err := info.Open()
	if err != nil {
		return err
	}
	defer from.Close()

	// Keep the executable bits but never write files others can modify.
	perm := info.Mode().Perm()&0o755 | 0o600
	to, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(to, from); err != nil {
		to.Close()
		return fmt.Errorf("writing %s: %w", abs, err)
	}
	return to.Close()
}
