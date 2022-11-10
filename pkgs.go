package devbox

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudflare/ahocorasick"
)

// packageNameRegex matches valid Nix package names and their base32 hash.
var packageNameRegex = regexp.MustCompile(`^([0-9abcdfghijklmnpqrsvwxyz]{32})-[^/]+$`)

// PackageStore is a file system containing packages that are compatible with
// Devbox.
type PackageStore struct {
	fs.FS
	path string
}

// LocalNixStore returns the default Nix store, which is typically /nix/store.
func LocalNixStore(path string) *PackageStore {
	return &PackageStore{
		FS:   os.DirFS(path),
		path: filepath.Clean(path),
	}
}

// Package retrieves a package by its name within the store. The name must be
// the fully unique directory name following the pattern <nixhash>-<name>. It
// may optionally include the /nix/store/ prefix.
func (p *PackageStore) Package(storeName string) (Package, error) {
	if strings.TrimSpace(storeName) == "" {
		return Package{}, fmt.Errorf("invalid package name %q: name is empty or whitespace", storeName)
	}
	cleaned := filepath.Clean(storeName)
	if relPath, err := filepath.Rel(p.path, storeName); err == nil {
		cleaned = relPath
	}
	if cleaned == "." {
		return Package{}, fmt.Errorf("invalid package name %q: name resolves to the package store root", storeName)
	}
	if !packageNameRegex.MatchString(cleaned) {
		return Package{}, fmt.Errorf("invalid package name %q: name doesn't match the regexp %#q", storeName, packageNameRegex)
	}
	pkgFS, err := fs.Sub(p, cleaned)
	if err != nil {
		return Package{}, fmt.Errorf("package store %q: unable to open package %q: %v", p.path, storeName, err)
	}
	pkg := Package{
		FS:        pkgFS,
		StoreName: cleaned,
	}
	pkg.DirectDependencies, err = p.directDependencies(pkg)
	if err != nil {
		return Package{}, fmt.Errorf("package store %q: unable to open package %q: cannot determine package dependencies: %v",
			p.path, storeName, err)
	}
	return pkg, nil
}

// directDependencies figures out the immediate dependencies for a given
// package.
func (p *PackageStore) directDependencies(pkg Package) ([]Package, error) {
	installedPkgs, installedHashes, err := p.installedPackages()
	if err != nil {
		return nil, err
	}
	scanner := newDependencyScanner(installedHashes)
	err = fs.WalkDir(pkg, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk package store: %v", err)
		}

		// Don't read directories or symlinks.
		if !d.Type().IsRegular() {
			return nil
		}
		f, err := pkg.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		var size int64
		if info, err := f.Stat(); err == nil {
			size = info.Size()
		}
		return scanner.scan(f, size)
	})
	if err != nil {
		return nil, err
	}

	deps := make([]Package, 0, 4)
	for i, found := range scanner.found {
		if found && installedPkgs[i].StoreName != pkg.StoreName {
			deps = append(deps, installedPkgs[i])
		}
	}
	return deps, err
}

// installedPackages returns slices of all the installed packages and their
// hashes. Each package in pkgs maps to its hash at the same index in hashes.
// Keeping the hashes in a separate slice makes it easier for a
// dependencyScanner to consume.
func (p *PackageStore) installedPackages() (pkgs []Package, hashes [][]byte, err error) {
	entries, err := fs.ReadDir(p, ".")
	if err != nil {
		return nil, nil, err
	}

	pkgs = make([]Package, 0, len(entries))
	hashes = make([][]byte, 0, len(entries))
	for _, file := range entries {
		if !file.IsDir() {
			continue
		}

		// Make sure that the directory name looks like a Nix package
		// and parse its hash.
		name := file.Name()
		matches := packageNameRegex.FindStringSubmatch(name)
		if len(matches) < 2 {
			continue
		}
		hashes = append(hashes, []byte(matches[1]))
		pkgFS, err := fs.Sub(p, name)
		if err != nil {
			return nil, nil, err
		}
		pkgs = append(pkgs, Package{
			FS:        pkgFS,
			StoreName: name,
		})
	}
	return pkgs, hashes, nil
}

// dependencyScanner scans through one or more readers for Nix base32 hashes.
type dependencyScanner struct {
	matcher *ahocorasick.Matcher
	buf     []byte

	installedHashes [][]byte
	found           []bool
}

// newDependencyScanner creates a dependencyScanner that looks for any of the
// installed hashes. When it finds a hash, it sets the corresponding index in
// the found field to true.
func newDependencyScanner(installedHashes [][]byte) dependencyScanner {
	scanner := dependencyScanner{
		buf:             make([]byte, os.Getpagesize()),
		installedHashes: installedHashes,
	}

	// Use [Aho–Corasick] because we need to look for every possible Nix
	// store hash at once, which can be a large list. It's about 5x faster
	// than using a regular expression and an order of magnitude faster than
	// bytes.Contains. This is the same algorithm that fgrep uses.
	//
	//   - bytes.Contains    = ~21s
	//   - regexp.FindAll    = ~2.5s
	//   - ahocorasick.Match = ~0.5s
	//
	// Optimization is warranted here because searching for hashes in large
	// packages with a naive approach can take considerable time. We might
	// want to add benchmarks here.
	//
	// [Aho–Corasick]: https://en.wikipedia.org/wiki/Aho–Corasick_algorithm>
	scanner.matcher = ahocorasick.NewMatcher(installedHashes)

	// index into storeHashes to track which ones we've found so far. This
	// is faster than a map since matcher gives us the indices, not the
	// actual matched substring.
	scanner.found = make([]bool, len(installedHashes))
	return scanner
}

// growBuffer grows the buffer to accommodate a read of a given size, unless
// that size exceeds the maximum buffer size limit. The buffer grows by powers
// of 2, so the new size might be greater than the requested size. After
// resizing, data in the old buffer is lost.
func (d *dependencyScanner) growBuffer(size int) {
	if size <= len(d.buf) {
		return
	}

	const maxBufSize = 2 << 24 // 32 MB
	if size > maxBufSize {
		size = maxBufSize
	}
	nextSize := len(d.buf)
	for nextSize < size {
		nextSize <<= 1
	}
	if nextSize > size {
		d.buf = make([]byte, nextSize)
	}
}

// scan reads from r looking for hashes until it encounters an error or EOF. If
// rsize is set to the size of the reader's data, scan will use it to more
// efficiently resize its read buffer.
func (d *dependencyScanner) scan(r io.Reader, rsize int64) error {
	d.growBuffer(int(rsize))

	const hashSize = 32 // bytes
	var (
		n   int
		err error
	)
	for err == nil {
		// The strategy here is to prefix each read buffer with the last
		// hashSize - 1 bytes of the previous read. This allows us to
		// detect hashes that might be split between two reads.
		n, err = r.Read(d.buf[hashSize-1:])
		if n == 0 {
			// Readers should generally block instead of returning
			// (n = 0, err = nil), but handle it just in case by
			// reading again.
			continue
		}

		n += hashSize - 1
		for _, matchIndex := range d.matcher.Match(d.buf[:n]) {
			d.found[matchIndex] = true
		}

		// Now copy over the end of this read to the beginning of the
		// buffer so it prefixes the next read.
		copy(d.buf, d.buf[len(d.buf)-hashSize-1:])
	}
	if err == io.EOF {
		return nil
	}
	return err
}

// Package is a file system that contains a package's files and metadata.
type Package struct {
	fs.FS

	// StoreName is the full name of the package within its store.
	StoreName string

	// DirectDependencies are the other packages in the store that this
	// package depends on. It does not contain transitive dependencies.
	DirectDependencies []Package
}
