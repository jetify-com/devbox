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

	"go.jetpack.io/devbox/internal/redact"
)

// Bytes returns a hex-encoded hash of b.
// TODO: This doesn't need to return an error.
func Bytes(b []byte) (string, error) {
	h := newHash()
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil)), nil
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
	return Bytes(b)
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
	return Bytes(buf.Bytes())
}

func newHash() hash.Hash { return sha256.New() }
