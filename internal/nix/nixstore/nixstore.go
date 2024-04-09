// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

// Package nixstore queries and resolves Nix store packages.
package nixstore

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// Root is the top-level directory of a Nix store. It maintains an index of
// packages for fast lookup and dependency resolution.
type Root struct {
	fs.FS

	// pkgs holds package information for every indexed package in the store.
	// Other fields point into this slice, so it should only be appended to and
	// not reordered.
	pkgs []Package

	// pkgByHash indexes packages by their store hash. Its entries point to the
	// packages in pkgs.
	pkgByHash map[string]*Package

	// storeHashes contains just package hashes for use with depScanner. Each hash
	// maps directly to the corresponding index in pkgs.
	storeHashes [][]byte

	// depScanner scans through the files in a package, looking for references to
	// other package hashes.
	depScanner dependencyScanner
}

// Local returns a local file system Nix store, which is typically /nix/store.
func Local(path string) (*Root, error) {
	return &Root{FS: newReadLinkFS(path)}, nil
}

// Package retrieves a package by its name within the store. The name must be
// the fully unique directory name following the pattern <nixhash>-<name>.
func (r *Root) Package(name string) (*Package, error) {
	cleaned := path.Clean(name)
	if cleaned == "." {
		return nil, errors.New("name is empty")
	}
	if strings.ContainsRune(cleaned, '/') {
		return nil, errors.New("name contains a '/'")
	}
	if !fs.ValidPath(name) {
		return nil, fmt.Errorf("invalid package name %q", name)
	}

	pkg := r.pkgByHash[name[:32]]
	if pkg == nil {
		if err := r.buildIndex(); err != nil {
			return nil, err
		}
		pkg = r.pkgByHash[name[:32]]
		if pkg == nil {
			return nil, fmt.Errorf("package not found: %s", name)
		}
	}
	return pkg, r.resolveDeps(pkg)
}

// indexPkg returns a [Package] with the given store name, adding it to the
// index if necessary. It assumes that the name is valid and in <hash>-<name>
// format.
func (r *Root) indexPkg(name string) (*Package, error) {
	hash := name[:32]
	if pkg := r.pkgByHash[hash]; pkg != nil {
		// We already have an instance of this package.
		return pkg, nil
	}

	// Add the package to end of the index slice and return a pointer to it.
	i := len(r.pkgs)
	r.pkgs = append(r.pkgs, Package{
		StoreName: name,
		Hash:      hash,
	})
	r.storeHashes = append(r.storeHashes, []byte(hash))
	pkg := &r.pkgs[i]

	var err error
	pkg.FS, err = fs.Sub(r, name)
	if err != nil {
		// Undo appending the new package before returning an error.
		r.pkgs = r.pkgs[:i]
		return nil, fmt.Errorf("unable to open package %s: %v", name, err)
	}
	r.pkgByHash[pkg.Hash] = pkg
	return pkg, nil
}

// buildIndex lists all of the files in the store root and indexes them by
// their hashes. This can be somewhat slow (1-2s) for a store with a lot of
// packages.
func (r *Root) buildIndex() error {
	entries, err := readDirUnsorted(r, ".")
	if err != nil {
		return fmt.Errorf("unable to list Nix store root directory: %w", err)
	}

	if r.pkgByHash == nil {
		r.pkgByHash = make(map[string]*Package, len(entries))
	}
	if cap(r.pkgs) < len(entries) {
		// Make sure we'll have enough capacity for the entries to avoid allocations.
		newCap := len(entries) + len(r.pkgs)

		pkgs := make([]Package, len(r.pkgs), newCap)
		copy(pkgs, r.pkgs)
		r.pkgs = pkgs

		hashes := make([][]byte, len(r.storeHashes), newCap)
		copy(hashes, r.storeHashes)
		r.storeHashes = hashes
	}

entries:
	for _, entry := range entries {
		// Skip hidden files or those that are too short to have a
		// base-32 hash prefix.
		name := entry.Name()
		if len(name) < 32 || name[0] == '.' {
			continue
		}

		// Make sure the hash is valid. Nix hashes must be alphanumeric without the
		// letters 'e', 'o', 't', or 'u'.
		hash := name[:32]
		for _, ch := range hash {
			switch {
			case '0' <= ch && ch <= '9':
			case 'a' <= ch && ch <= 'z':
				switch ch {
				case 'e', 'o', 't', 'u':
					continue entries
				}
			default:
				continue entries
			}
		}
		if _, err := r.indexPkg(name); err != nil {
			return err
		}
	}
	r.depScanner = newDependencyScanner(r.storeHashes)
	return nil
}

// resolveDeps populates the direct dependencies of pkg.
func (r *Root) resolveDeps(pkg *Package) error {
	if pkg.DirectDependencies != nil {
		// Already resolved.
		return nil
	}

	// Find dependencies by looking at every file in the package to see if it
	// references another package's hash.
	foundDeps := map[*Package]struct{}{}
	err := fs.WalkDir(pkg, ".", func(entryPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Scan the contents of regular files for references to other packages.
		if entry.Type().IsRegular() {
			f, err := pkg.Open(entryPath)
			if err != nil {
				return err
			}
			matches, err := r.depScanner.scan(f)
			for _, matchIndex := range matches {
				foundDeps[&r.pkgs[matchIndex]] = struct{}{}
			}
			f.Close()
			return err
		}

		// Look at the destination of symlinks to see if they point to another package.
		if entry.Type() == fs.ModeSymlink {
			src := path.Join(pkg.StoreName, entryPath)
			dst, err := readLink(r.FS, src)
			if err != nil {
				return err
			}
			if len(dst) > 32 {
				dep := r.pkgByHash[dst[:32]]
				if dep == nil {
					return fmt.Errorf("symlink at %s points to a missing package: %s", src, dst)
				}
				foundDeps[dep] = struct{}{}
			}
			return nil
		}

		// Ignore all other file types.
		return nil
	})
	if err != nil {
		return fmt.Errorf("error scanning %s for dependencies: %v", pkg.StoreName, err)
	}

	// Recursively resolve the found dependencies to build up a DAG of packages.
	pkg.DirectDependencies = make([]*Package, 0, len(foundDeps))
	for dep := range foundDeps {
		if dep == pkg {
			// Skip self-references.
			continue
		}
		if err := r.resolveDeps(dep); err != nil {
			return err
		}
		pkg.DirectDependencies = append(pkg.DirectDependencies, dep)
	}
	return nil
}

// Package is a file system that contains a package's files and metadata.
type Package struct {
	fs.FS

	// StoreName is the full name of the package within its store.
	StoreName string

	// Hash is the package's base-32 Nix store hash.
	Hash string

	// DirectDependencies are the other packages in the store that this
	// package depends on. It does not contain transitive dependencies.
	DirectDependencies []*Package
}

func (p Package) String() string {
	return p.StoreName
}

// TopologicalSort resolves the dependency tree for a package and returns it as
// a slice of packages in topological order.
func TopologicalSort(pkg *Package) []*Package {
	pkgs := make([]*Package, 0, len(pkg.DirectDependencies))
	seen := make(map[*Package]bool, len(pkg.DirectDependencies))
	return tsort(pkgs, seen, pkg)
}

func tsort(sorted []*Package, seen map[*Package]bool, pkg *Package) []*Package {
	if seen[pkg] {
		return sorted
	}
	for _, dep := range pkg.DirectDependencies {
		sorted = tsort(sorted, seen, dep)
	}
	seen[pkg] = true
	return append(sorted, pkg)
}
