// Package nixstore queries and resolves Nix store packages.
package nixstore

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/nix-community/go-nix/pkg/nar"
	"github.com/nix-community/go-nix/pkg/narinfo"
	"github.com/ulikunitz/xz"
	"golang.org/x/sys/unix"
)

// Root is the top-level directory of a Nix store. It maintains an index of
// packages for fast lookup and dependency resolution.
type Root struct {
	fs.FS

	// osPath is the OS-specific path for local Nix stores (as opposed to fs
	// paths). This path is necessary for write operations. Whenever possible
	// prefer using the fs.FS.
	osPath string

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
	root := &Root{
		FS:     newReadLinkFS(path),
		osPath: filepath.Clean(path),
	}
	return root, root.buildIndex()
}

// Remote returns a remote Nix store, such as a binary cache.
func Remote(url string) (*Root, error) {
	fsys, err := NewS3BucketFS(url)
	if err != nil {
		return nil, err
	}
	return &Root{
		FS:        fsys,
		pkgByHash: make(map[string]*Package),
	}, nil
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
	pkg, err := r.indexPkg(name)
	if err != nil {
		return nil, err
	}
	return pkg, r.resolveDeps(pkg)
}

func (r *Root) PackageAttrPath(attr string) (*Package, error) {
	if pkgByAttrPath == nil {
		buildSearchIndex()
	}
	storePath := pkgByAttrPath[attr].Out
	if storePath == "" {
		return nil, errors.New("package not found")
	}

	// The store path from the search index will be the absolute path
	// with a /nix/store/ prefix. We need to get the base name for
	// r.Package.
	return r.Package(filepath.Base(storePath))
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
		store:     r,
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
//
//nolint:revive
func (r *Root) resolveDeps(pkg *Package) error {
	if pkg.DirectDependencies != nil {
		// Already resolved.
		return nil
	}

	// If there's a narinfo available for this package then we don't need to scan
	// for dependencies.
	narinfoPath := pkg.Hash + ".narinfo"
	f, err := r.Open(narinfoPath)
	if err == nil {
		defer f.Close()
		ni, err := narinfo.Parse(f)
		if err != nil {
			return err
		}
		pkg.narPath = ni.URL
		pkg.narCompression = ni.Compression
		for _, ref := range ni.References {
			dep, err := r.indexPkg(ref)
			if err != nil {
				return err
			}
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

	// Find dependencies by looking at every file in the package to see if it
	// references another package's hash.
	foundDeps := map[*Package]struct{}{}
	err = fs.WalkDir(pkg, ".", func(entryPath string, entry fs.DirEntry, err error) error {
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

func (r *Root) Install(pkg *Package) error {
	for _, pkg := range topologicalSort(pkg) {
		if local, err := r.Package(pkg.StoreName); err == nil {
			if _, err := fs.Stat(local, "."); err == nil {
				continue
			}
		}

		var (
			narFile io.ReadCloser
			err     error
		)
		narFile, err = pkg.store.Open(pkg.narPath)
		if err != nil {
			return fmt.Errorf("unable to open nar file: %v", err)
		}
		defer narFile.Close()

		if pkg.narCompression == "xz" {
			xzr, err := xz.NewReader(narFile)
			if err != nil {
				return fmt.Errorf("nar file %s is not a valid xz archive: %v",
					pkg.narPath, err)
			}
			narFile = io.NopCloser(xzr)
		}
		narr, err := nar.NewReader(narFile)
		if err != nil {
			return fmt.Errorf("nar file %s is invalid: %w", pkg.narPath, err)
		}
		defer narr.Close()

		// First copy to a staging directory and then move it when done. This reduces
		// the odds of ending up with a partial install if there's an error. Put the
		// temp directory in the store so they're on the same volume and a copy won't
		// be needed.
		storeTemp := filepath.Join(r.osPath, ".devbox")
		if err := os.MkdirAll(storeTemp, 0777); err != nil {
			return fmt.Errorf("create .devbox staging dir: %v", err)
		}
		dir, err := os.MkdirTemp(storeTemp, pkg.StoreName+"-")
		if err != nil {
			return fmt.Errorf("create temp dir: %v", err)
		}
		err = extractNar(narr, dir)
		if err != nil {
			return err
		}
		storePath := filepath.Join(r.osPath, pkg.StoreName)
		if err := os.Rename(dir, storePath); err != nil {
			return err
		}
		fmt.Println("Installed", storePath)
	}
	return nil
}

// TODO(gcurtis): we're just using the default nixbld group ID.
// Fix this to be smarter about getting the actual gid.
const nixgid = 30000

func extractNar(narr *nar.Reader, dir string) error {
	for {
		header, err := narr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading next file in nar archive: %v", err)
		}

		dst := filepath.Join(dir, header.Path)
		if dst == dir {
			continue
		}
		switch header.Type {
		case nar.TypeRegular:
			err := extractFile(narr, dst, header.Size, header.Executable)
			if err != nil {
				return fmt.Errorf("extract regular file %s: %v", dst, err)
			}
		case nar.TypeDirectory:
			err := extractDir(dst)
			if err != nil {
				return fmt.Errorf("extract directory %s: %v", dst, err)
			}
		case nar.TypeSymlink:
			err := extractLink(header.LinkTarget, dst, header.Executable)
			if err != nil {
				return fmt.Errorf("extract symlink %s to target %s: %v",
					header.LinkTarget, dst, err)
			}
		}
	}
	return nil
}

func extractFile(src io.Reader, dstPath string, size int64, exe bool) (err error) {
	perm := fs.FileMode(0444)
	if exe {
		perm = 0555
	}
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := dst.Close()
		if err != nil {
			return
		}
		if closeErr != nil {
			err = closeErr
			return
		}
		timeErr := os.Chtimes(dstPath, time.UnixMilli(0), time.UnixMilli(0))
		if timeErr != nil {
			err = timeErr
			return
		}
	}()

	if err := dst.Truncate(size); err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	if err := os.Chown(dstPath, os.Getuid(), nixgid); err != nil {
		return err
	}
	return dst.Chmod(perm)
}

func extractDir(dstPath string) error {
	err := os.Mkdir(dstPath, 0755)
	if err != nil {
		return err
	}
	if err := os.Chown(dstPath, os.Getuid(), nixgid); err != nil {
		return err
	}
	return os.Chtimes(dstPath, time.UnixMilli(0), time.UnixMilli(0))
}

func extractLink(oldname, newname string, exe bool) error {
	err := os.Symlink(oldname, newname)
	if err != nil {
		return err
	}
	if err := os.Lchown(newname, os.Getuid(), nixgid); err != nil {
		return err
	}

	perm := uint32(0444)
	if exe {
		perm = 0555
	}
	if err := unix.Fchmodat(unix.AT_FDCWD, newname, perm, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return err
	}
	return unix.Lutimes(newname, make([]unix.Timeval, 2))
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

	// narPath is the path to a nar file containing this package when one
	// is available. If the store doesn't have a nar for this package then narPath
	// will be empty.
	narPath        string
	narCompression string

	store *Root
}

func (p Package) String() string {
	return p.StoreName
}

// topologicalSort resolves the dependency tree for a package and returns it as
// a slice of packages in topological order.
func topologicalSort(pkg *Package) []*Package {
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
