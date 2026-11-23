// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"archive/tar"
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
// Every entry is confined to dest, including through symlinks that earlier
// entries created: parent directories are resolved on disk and must stay
// inside dest, symlink targets must be relative and resolve inside dest,
// hard links must point at an earlier entry, and an existing symlink is
// never followed when writing a file.
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

	// Work on the real path of dest so that containment checks compare
	// resolved paths with resolved paths (dest may live under a symlink,
	// e.g. /var -> /private/var on macOS).
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	root, err := filepath.EvalSymlinks(dest)
	if err != nil {
		return err
	}

	handler := func(_ context.Context, info archives.FileInfo) error {
		rel, ok := stripPath(info.NameInArchive, strip)
		if !ok {
			return nil
		}
		if !isInside(root, filepath.Join(root, rel)) {
			return fmt.Errorf("archive entry %q escapes the destination", info.NameInArchive)
		}

		mode := info.Mode()
		switch {
		case isHardLink(info):
			return writeHardLink(root, rel, info.LinkTarget, strip, info.NameInArchive)
		case mode.IsDir():
			_, err := mkdirInside(root, rel, info.NameInArchive)
			return err
		case mode&os.ModeSymlink != 0:
			return writeSymlink(root, rel, info.LinkTarget, info.NameInArchive)
		case mode.IsRegular():
			return writeRegular(root, rel, info)
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

// mkdirInside makes sure the directory root/rel exists and returns its
// real path. Missing components are created. An existing symlink component
// is followed only when it resolves to a directory inside root, so a link
// planted by an earlier entry can never redirect a write outside of it.
func mkdirInside(root, rel, name string) (string, error) {
	cur := root
	if rel == "" || rel == "." {
		return cur, nil
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		next := filepath.Join(cur, part)
		fi, err := os.Lstat(next)
		switch {
		case errors.Is(err, os.ErrNotExist):
			if err := os.Mkdir(next, 0o755); err != nil {
				return "", err
			}
		case err != nil:
			return "", err
		case fi.Mode()&os.ModeSymlink != 0:
			resolved, err := filepath.EvalSymlinks(next)
			if err != nil {
				return "", fmt.Errorf("archive entry %q: %w", name, err)
			}
			if !isInside(root, resolved) {
				return "", fmt.Errorf("archive entry %q escapes the destination through a symlink", name)
			}
			if fi, err := os.Stat(resolved); err != nil || !fi.IsDir() {
				return "", fmt.Errorf("archive entry %q: %s is not a directory", name, part)
			}
			next = resolved
		case !fi.IsDir():
			return "", fmt.Errorf("archive entry %q: %s is not a directory", name, part)
		}
		cur = next
	}
	return cur, nil
}

// removeNonDir removes whatever non-directory is at p, so that a new entry
// replaces it instead of writing through it (a symlink) or into an inode
// shared with another name (a hard link).
func removeNonDir(p string) error {
	fi, err := os.Lstat(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return nil
	}
	return os.Remove(p)
}

// writeSymlink creates the symlink root/rel -> target. The target must be
// relative, and its leading ".." components (the only ones allowed: a ".."
// after a name would be resolved relative to wherever that name, possibly
// a symlink, points) may not climb out of root from the link's real parent
// directory. Together with mkdirInside this keeps every symlink in the
// tree resolving inside root, whatever order the entries come in.
func writeSymlink(root, rel, target, name string) error {
	if target == "" || filepath.IsAbs(target) || strings.HasPrefix(target, "/") {
		return fmt.Errorf("archive entry %q: unsupported symlink target %q", name, target)
	}
	up := 0
	seenName := false
	for _, part := range strings.Split(filepath.ToSlash(target), "/") {
		switch part {
		case "", ".":
		case "..":
			if seenName {
				return fmt.Errorf("archive entry %q: unsupported symlink target %q", name, target)
			}
			up++
		default:
			seenName = true
		}
	}

	parent, err := mkdirInside(root, filepath.Dir(rel), name)
	if err != nil {
		return err
	}
	base := parent
	for range up {
		base = filepath.Dir(base)
	}
	if !isInside(root, base) {
		return fmt.Errorf("archive entry %q: symlink target %q escapes the destination", name, target)
	}

	abs := filepath.Join(parent, filepath.Base(rel))
	if err := os.Remove(abs); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Symlink(filepath.FromSlash(target), abs)
}

// isHardLink reports whether info is a tar hard link entry. archive/tar
// reports those as regular files of size 0.
func isHardLink(info archives.FileInfo) bool {
	hdr, ok := info.Header.(*tar.Header)
	return ok && hdr.Typeflag == tar.TypeLink
}

// writeHardLink makes root/rel a hard link to (or, where linking fails, a
// copy of) an earlier entry. linkname is a path in the archive, so it gets
// the same strip_components treatment as entry names. The target must be
// an existing regular file inside root.
func writeHardLink(root, rel, linkname string, strip int, name string) error {
	targetRel, ok := stripPath(linkname, strip)
	if !ok {
		return fmt.Errorf("archive entry %q: hard link target %q is not extracted", name, linkname)
	}
	target, err := filepath.EvalSymlinks(filepath.Join(root, targetRel))
	if err != nil {
		return fmt.Errorf("archive entry %q: hard link target %q: %w", name, linkname, err)
	}
	if !isInside(root, target) {
		return fmt.Errorf("archive entry %q: hard link target %q escapes the destination", name, linkname)
	}
	if fi, err := os.Lstat(target); err != nil || !fi.Mode().IsRegular() {
		return fmt.Errorf("archive entry %q: hard link target %q is not a regular file", name, linkname)
	}

	parent, err := mkdirInside(root, filepath.Dir(rel), name)
	if err != nil {
		return err
	}
	abs := filepath.Join(parent, filepath.Base(rel))
	if abs == target {
		return nil
	}
	if err := removeNonDir(abs); err != nil {
		return err
	}
	if err := os.Link(target, abs); err == nil {
		return nil
	}
	fi, err := os.Stat(target)
	if err != nil {
		return err
	}
	return copyFile(target, abs, fi.Mode().Perm())
}

func writeRegular(root, rel string, info archives.FileInfo) error {
	parent, err := mkdirInside(root, filepath.Dir(rel), info.NameInArchive)
	if err != nil {
		return err
	}
	abs := filepath.Join(parent, filepath.Base(rel))
	// Never write through a symlink that an earlier entry planted.
	if err := removeNonDir(abs); err != nil {
		return err
	}

	from, err := info.Open()
	if err != nil {
		return err
	}
	defer from.Close()

	// Keep the executable bits but never write files others can modify.
	perm := info.Mode().Perm()&0o755 | 0o600
	return writeFile(abs, from, perm)
}

func copyFile(src, dest string, perm os.FileMode) error {
	from, err := os.Open(src)
	if err != nil {
		return err
	}
	defer from.Close()
	return writeFile(dest, from, perm)
}

func writeFile(abs string, from io.Reader, perm os.FileMode) error {
	to, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(to, from); err != nil {
		to.Close()
		return fmt.Errorf("writing %s: %w", abs, err)
	}
	return to.Close()
}
