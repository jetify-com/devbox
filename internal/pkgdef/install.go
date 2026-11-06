// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// ErrUnsupportedSystem is returned when a definition does not list the
// current system in its platforms.
var ErrUnsupportedSystem = errors.New("package does not support this system")

// ErrChecksumMismatch is returned when a downloaded artifact does not match
// the sha256 recorded in the lockfile.
var ErrChecksumMismatch = errors.New("checksum mismatch")

// markerFile is written into an install directory once the install has
// fully completed, including post_install hooks. A directory without it is
// a failed or interrupted install and gets redone.
const markerFile = ".devbox-pkgdef.json"

// marker is the content of markerFile.
type marker struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	System  string `json:"system"`
	URL     string `json:"url,omitempty"`
	Sha256  string `json:"sha256,omitempty"`
}

// Installed describes a completed install.
type Installed struct {
	// Dir is the install root.
	Dir string

	// Sha256 is the digest of the artifact that was installed. When the
	// lockfile had no digest (checksum type "none"), callers should record
	// this one.
	Sha256 string

	// BinDirs are the absolute directories to add to PATH.
	BinDirs []string
}

// Install makes version of def available on system and returns where it
// landed. It is idempotent: a completed install is returned as is without
// any network access. Concurrent installs of the same package (from other
// processes too) are serialized with a file lock.
//
// artifact.URL may be empty, in which case the URL is rendered from the
// definition. When artifact.Sha256 is set the download must match it.
func (c *Client) Install(ctx context.Context, def *Definition, version, system string, artifact Artifact) (*Installed, error) {
	if !def.SupportsSystem(system) {
		return nil, fmt.Errorf("%s@%s: %w (%s; supported: %s)",
			def.Name, version, ErrUnsupportedSystem, system, strings.Join(def.Platforms, ", "))
	}
	dir := c.InstallDir(def.Name, version, system)

	// Fast path: no lock needed to read a finished install.
	if m, ok := readMarker(dir); ok {
		return c.installedFromMarker(def, version, dir, m, artifact)
	}

	unlock, err := lockDir(dir)
	if err != nil {
		return nil, err
	}
	defer unlock()

	// Another process may have finished while we waited for the lock.
	if m, ok := readMarker(dir); ok {
		return c.installedFromMarker(def, version, dir, m, artifact)
	}
	return c.install(ctx, def, version, system, dir, artifact)
}

func (c *Client) installedFromMarker(def *Definition, version, dir string, m marker, artifact Artifact) (*Installed, error) {
	if artifact.Sha256 != "" && m.Sha256 != "" && !strings.EqualFold(artifact.Sha256, m.Sha256) {
		return nil, fmt.Errorf("%s@%s: %w: %s was installed from an artifact with sha256 %s but the lockfile expects %s; remove the directory to reinstall",
			def.Name, version, ErrChecksumMismatch, dir, m.Sha256, artifact.Sha256)
	}
	return c.installed(def, dir, m.Sha256), nil
}

func (c *Client) install(ctx context.Context, def *Definition, version, system, dir string, artifact Artifact) (*Installed, error) {
	install := def.InstallFor(system)
	vars, err := def.TemplateVars(system, version)
	if err != nil {
		return nil, err
	}
	vars.InstallDir = dir
	vars.CacheDir = c.CacheDir

	switch install.Type {
	case InstallArchive, InstallBinary:
	case InstallManager, InstallScript:
		return nil, fmt.Errorf("%s: install.type %q: %w", def.Name, install.Type, ErrNotImplemented)
	default:
		return nil, fmt.Errorf("%s: install.type %q is not supported", def.Name, install.Type)
	}

	if artifact.URL == "" {
		artifact.URL, err = Render(install.URL, vars)
		if err != nil {
			return nil, err
		}
	}
	vars = vars.withURL(artifact.URL)

	// Download next to the install dir so that a rename into place stays on
	// one filesystem.
	downloadDir := filepath.Join(c.CacheDir, ".downloads", def.Name, version, system)
	archivePath := filepath.Join(downloadDir, vars.Filename)
	sha, err := c.download(ctx, artifact.URL, archivePath)
	if err != nil {
		return nil, fmt.Errorf("%s@%s: %w", def.Name, version, err)
	}
	if artifact.Sha256 != "" && !strings.EqualFold(sha, artifact.Sha256) {
		_ = os.Remove(archivePath)
		return nil, fmt.Errorf("%s@%s: %w: %s has sha256 %s, lockfile expects %s",
			def.Name, version, ErrChecksumMismatch, artifact.URL, sha, artifact.Sha256)
	}

	// Unpack into a staging directory and rename it into place so that a
	// crash never leaves a half-populated install dir behind.
	if err := os.RemoveAll(dir); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return nil, err
	}
	staging, err := os.MkdirTemp(filepath.Dir(dir), filepath.Base(dir)+".tmp-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(staging)
	// MkdirTemp hard-codes 0700; the install is meant to be shareable like
	// its parents.
	if err := os.Chmod(staging, 0o755); err != nil {
		return nil, err
	}

	switch install.Type {
	case InstallArchive:
		err = extractArchive(ctx, archivePath, staging, install.StripComponents)
	case InstallBinary:
		binName := install.BinName
		if binName == "" {
			binName = def.Name
		}
		err = placeBinary(archivePath, filepath.Join(staging, "bin", binName))
	}
	if err != nil {
		return nil, fmt.Errorf("%s@%s: unpacking %s: %w", def.Name, version, vars.Filename, err)
	}
	_ = os.Remove(archivePath)

	if err := os.Rename(staging, dir); err != nil {
		return nil, err
	}

	if err := c.runPostInstall(ctx, def, dir, vars); err != nil {
		// Leave the directory in place for debugging, but without a marker
		// so the next install retries.
		return nil, fmt.Errorf("%s@%s: %w", def.Name, version, err)
	}

	m := marker{Name: def.Name, Version: version, System: system, URL: artifact.URL, Sha256: sha}
	if err := writeMarker(dir, m); err != nil {
		return nil, err
	}
	return c.installed(def, dir, sha), nil
}

// IsInstalled reports whether version of def is fully installed on system.
func (c *Client) IsInstalled(def *Definition, version, system string) bool {
	_, ok := readMarker(c.InstallDir(def.Name, version, system))
	return ok
}

func (c *Client) installed(def *Definition, dir, sha string) *Installed {
	bins := make([]string, 0, len(def.BinDirs()))
	for _, b := range def.BinDirs() {
		bins = append(bins, filepath.Join(dir, b))
	}
	return &Installed{Dir: dir, Sha256: sha, BinDirs: bins}
}

// lockDir takes an exclusive advisory lock on <dir>.lock so that only one
// process installs into dir at a time. The returned function releases it.
func lockDir(dir string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return nil, err
	}
	lockFile, err := os.OpenFile(dir+".lock", os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		lockFile.Close()
		return nil, fmt.Errorf("locking %s: %w", dir, err)
	}
	return func() {
		_ = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
		lockFile.Close()
	}, nil
}

// download fetches rawURL to dest (unless dest already exists) and returns
// the hex sha256 of the file.
func (c *Client) download(ctx context.Context, rawURL, dest string) (string, error) {
	if sha, err := fileSHA256(dest); err == nil {
		return sha, nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}

	body, err := c.open(ctx, rawURL, "")
	if err != nil {
		return "", err
	}
	defer body.Close()

	tmp, err := os.CreateTemp(filepath.Dir(dest), filepath.Base(dest)+".part-")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())

	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, hasher), body); err != nil {
		tmp.Close()
		return "", fmt.Errorf("downloading %s: %w", rawURL, err)
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmp.Name(), dest); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func placeBinary(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// hookOutputLimit bounds how much hook output is kept for error messages.
const hookOutputLimit = 4096

func (c *Client) runPostInstall(ctx context.Context, def *Definition, dir string, vars Vars) error {
	for _, hook := range def.Hooks.PostInstall {
		script, err := Render(hook, vars)
		if err != nil {
			return err
		}
		// Keep the tail of the output so a failure is diagnosable even when
		// Stderr is discarded.
		tail := &tailBuffer{limit: hookOutputLimit}
		output := io.MultiWriter(c.Stderr, tail)

		cmd := exec.CommandContext(ctx, "sh", "-c", script)
		cmd.Dir = dir
		cmd.Stdout = output
		cmd.Stderr = output
		cmd.Env = append(os.Environ(),
			"DEVBOX_PKG_INSTALL_DIR="+dir,
			"DEVBOX_PKG_VERSION="+vars.Version,
			"DEVBOX_PKG_SYSTEM="+vars.System,
		)
		if err := cmd.Run(); err != nil {
			if out := strings.TrimSpace(tail.String()); out != "" {
				return fmt.Errorf("post_install hook %q: %w\n%s", hook, err, out)
			}
			return fmt.Errorf("post_install hook %q: %w", hook, err)
		}
	}
	return nil
}

// tailBuffer keeps the last limit bytes written to it.
type tailBuffer struct {
	buf   bytes.Buffer
	limit int
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.buf.Write(p)
	if t.buf.Len() > t.limit {
		excess := t.buf.Len() - t.limit
		t.buf.Next(excess)
	}
	return len(p), nil
}

func (t *tailBuffer) String() string {
	return t.buf.String()
}

func readMarker(dir string) (marker, bool) {
	b, err := os.ReadFile(filepath.Join(dir, markerFile))
	if err != nil {
		return marker{}, false
	}
	var m marker
	if err := json.Unmarshal(b, &m); err != nil {
		return marker{}, false
	}
	return m, true
}

func writeMarker(dir string, m marker) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, markerFile), b, 0o644)
}
