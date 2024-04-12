// Package cachehash generates non-cryptographic cache keys.
//
// The functions in this package make no guarantees about the underlying hashing
// algorithm. It should only be used for caching, where it's ok if the hash for
// a given input changes.
package cachehash

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"hash"
	"io"
	"os"
	"strings"

	"github.com/gosimple/slug"
	"go.jetpack.io/devbox/internal/redact"
)

// Bytes returns a hex-encoded hash of b.
func Bytes(b []byte) string {
	h := newHash()
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}

// Bytes6 returns the first 6 characters of the hash of b.
func Bytes6(b []byte) string {
	hash := Bytes(b)
	return hash[:min(len(hash), 6)]
}

// File returns a hex-encoded hash of a file's contents.
func File(path string) (string, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := newHash()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// JSON marshals a to JSON and returns its hex-encoded hash.
func JSON(a any) (string, error) {
	b, err := json.Marshal(a)
	if err != nil {
		return "", redact.Errorf("marshal to json for hashing: %v", err)
	}
	return Bytes(b), nil
}

// JSONFile compacts the JSON in a file and returns its hex-encoded hash.
func JSONFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}
	if err := json.Compact(buf, b); err != nil {
		return "", redact.Errorf("compact json for hashing: %v", err)
	}
	return Bytes(buf.Bytes()), nil
}

func newHash() hash.Hash { return sha256.New() }

// Slug returns a deterministic URL slug version of the string. With the
// following characteristics:
//
// * A 6 character hash is appended to avoid collisions
// * Trims the slug to last 50 characters
// * Removes any prefixed hashes to ensure first character is alphanumeric
func Slug(s string) string {
	s = slug.Make(s) + "-" + Bytes6([]byte(s))
	return strings.TrimPrefix(s[max(0, len(s)-50):], "-")
}
