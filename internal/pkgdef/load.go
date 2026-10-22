// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrDefinitionChanged is returned when a remote definition no longer
// matches the hash pinned in the lockfile.
var ErrDefinitionChanged = errors.New("package definition changed")

// Loaded is a definition together with where it came from.
type Loaded struct {
	Ref        Ref
	Definition *Definition

	// Content is the raw file content and Hash its hex sha256, which is
	// what devbox.lock records as definition_hash. It is a plain sha256
	// (not cachehash, whose algorithm may change) because the lockfile
	// relies on it staying stable across Devbox versions.
	Content []byte
	Hash    string

	// Dir is the directory containing the definition file for local
	// definitions. create_files sources are resolved relative to it. It
	// is empty for remote definitions.
	Dir string
}

// Load reads and parses the definition that ref points at. Relative paths
// are resolved against projectDir.
//
// pinnedHash is the definition_hash from the lockfile, or "". For remote
// definitions a pinned hash is served from the local cache without any
// network access, and a freshly fetched definition that does not match it
// is rejected with ErrDefinitionChanged. Local definitions ignore it: the
// file on disk is the source of truth and edits are expected.
func (c *Client) Load(ctx context.Context, ref Ref, projectDir, pinnedHash string) (*Loaded, error) {
	if ref.IsLocal() {
		return loadLocal(ref, projectDir)
	}
	return c.loadRemote(ctx, ref, pinnedHash)
}

// LocalPath returns the absolute path of a local definition. It returns ""
// for remote definitions.
func (r Ref) LocalPath(projectDir string) string {
	if !r.IsLocal() {
		return ""
	}
	p := strings.TrimPrefix(r.Source, "file://")
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(projectDir, p)
}

func loadLocal(ref Ref, projectDir string) (*Loaded, error) {
	path := ref.LocalPath(projectDir)
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("package definition %s: %w", ref.Source, os.ErrNotExist)
	}
	if err != nil {
		return nil, err
	}
	def, err := Parse(content)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &Loaded{
		Ref:        ref,
		Definition: def,
		Content:    content,
		Hash:       contentHash(content),
		Dir:        filepath.Dir(path),
	}, nil
}

func (c *Client) loadRemote(ctx context.Context, ref Ref, pinnedHash string) (*Loaded, error) {
	if pinnedHash != "" {
		if content, err := os.ReadFile(c.definitionCachePath(pinnedHash)); err == nil {
			if contentHash(content) == pinnedHash {
				return c.parseRemote(ref, content)
			}
		}
	}

	content, err := c.get(ctx, ref.Source)
	if err != nil {
		return nil, fmt.Errorf("package definition %s: %w", ref.Source, err)
	}
	hash := contentHash(content)
	if pinnedHash != "" && hash != pinnedHash {
		return nil, fmt.Errorf("package definition %s: %w: devbox.lock pins %s but the server now serves %s; run `devbox update` to accept the new definition",
			ref.Source, ErrDefinitionChanged, pinnedHash, hash)
	}

	loaded, err := c.parseRemote(ref, content)
	if err != nil {
		return nil, err
	}
	// Cache by content hash so future loads pinned to this hash are
	// offline. Failing to cache is not fatal.
	cachePath := c.definitionCachePath(hash)
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err == nil {
		_ = os.WriteFile(cachePath, content, 0o644)
	}
	return loaded, nil
}

func (c *Client) parseRemote(ref Ref, content []byte) (*Loaded, error) {
	def, err := Parse(content)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ref.Source, err)
	}
	return &Loaded{
		Ref:        ref,
		Definition: def,
		Content:    content,
		Hash:       contentHash(content),
	}, nil
}

func contentHash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func (c *Client) definitionCachePath(hash string) string {
	return filepath.Join(c.CacheDir, ".definitions", hash+".json")
}
