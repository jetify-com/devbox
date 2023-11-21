// Package cachehash generates non-cryptographic cache keys.
//
// The functions in this package make no guarantees about the underlying hashing
// algorithm. It should only be used for caching, where it's ok if the hash for
// a given input changes.
package cachehash

import (
	"encoding/hex"
	"errors"
	"hash"
	"hash/fnv"
	"io"
	"os"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"go.jetpack.io/devbox/internal/redact"
)

// Bytes returns a hex-encoded hash of b.
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

// JSON marshals a to canonical JSON and returns its hex-encoded hash.
func JSON(a any) (string, error) {
	b, err := json.Marshal(a)
	if err != nil {
		return "", redact.Errorf("marshal to json for hashing: %v", err)
	}
	return JSONBytes(b)
}

// JSONBytes canonicalizes the raw JSON bytes in b and returns its hex-encoded
// hash. It modifies b directly. To preserve the original JSON, make a copy of
// of it before calling JSONBytes.
func JSONBytes(b []byte) (string, error) {
	v := jsontext.Value(b)
	if err := v.Canonicalize(); err != nil {
		return "", redact.Errorf("canonicalize json for hashing: %v", err)
	}
	return Bytes(v)
}

// JSONFile canonicalizes the JSON in a file and returns its hex-encoded hash.
func JSONFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return JSONBytes(b)
}

func newHash() hash.Hash { return fnv.New64a() }
