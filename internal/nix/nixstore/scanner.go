// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nixstore

import (
	"io"

	"github.com/cloudflare/ahocorasick"
)

// dependencyScanner scans through one or more readers for Nix base-32 hashes.
type dependencyScanner struct {
	// matcher uses [Aho–Corasick] to look for every possible Nix
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
	matcher *ahocorasick.Matcher

	// buf is a reusable buffer for reading file contents.
	buf []byte

	// matches contains indexes into the slice given to
	// newDependencyScanner for each match. For example, if the
	// storeHashes slice is ["a", "b", "c"] and matches is [2, 0], then that
	// means "c" and "a" were found. It may contain duplicate indexes.
	matches []int
}

// newDependencyScanner creates a dependencyScanner that looks for a set of
// store hashes.
func newDependencyScanner(storeHashes [][]byte) dependencyScanner {
	return dependencyScanner{
		matcher: ahocorasick.NewMatcher(storeHashes),

		// The buffer should be large enough to hold smaller files in a single read.
		// If it's too small, then reading through all the files in a package can
		// take a long time.
		buf: make([]byte, 2<<24), // 16 MiB

		// Most packages won't have more than 256 non-unique references to other
		// packages, and we want to avoid reallocating while scanning.
		matches: make([]int, 0, 256),
	}
}

// scan reads from r looking for hashes until it encounters an error or
// [io.EOF]. It returns a slice of storeHashes indexes (as passed to
// newDependencyScanner) to indicate which hashes it found. The slice is only
// valid until the next scan and may contains duplicate indexes.
func (d *dependencyScanner) scan(r io.Reader) (indexes []int, err error) {
	const hashSize = 32 // bytes
	n := 0
	d.matches = d.matches[:0]
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
		d.matches = append(d.matches, d.matcher.Match(d.buf[:n])...)

		// Now copy over the end of this read to the beginning of the
		// buffer so it prefixes the next read.
		copy(d.buf, d.buf[len(d.buf)-hashSize-1:])
	}
	if err == io.EOF {
		return d.matches, nil
	}
	return nil, err
}
