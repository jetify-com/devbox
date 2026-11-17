// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/sha512"
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
	Sha512  string `json:"sha512,omitempty"`

	// InstallHash is Definition.InstallFingerprint at install time. When
	// the definition's install recipe changes, the install is redone.
	InstallHash string `json:"install_hash,omitempty"`
}

// Installed describes a completed install.
type Installed struct {
	// Dir is the install root.
	Dir string

	// URL is the artifact the install came from.
	URL string

	// Sha256 is the digest of the artifact that was installed. When the
	// lockfile had no digest (checksum type "none"), callers should record
	// this one.
	Sha256 string

	// BinDirs are the absolute directories to add to PATH.
	BinDirs []string
}

// Install makes version of def available on system and returns where it
// landed. It is idempotent: a completed install from the same recipe is
// returned as is without any network access; an install made from a
// definition whose install recipe has since changed is redone. Concurrent
// installs of the same package (from other processes too) are serialized
// with a file lock.
//
// artifact.URL may be empty, in which case the URL is rendered from the
// definition (or looked up in its index). Relative file:// URLs are
// resolved against the definition's directory. When the artifact has a
// digest the download must match it.
func (c *Client) Install(ctx context.Context, def *Definition, version, system string, artifact Artifact) (*Installed, error) {
	if !def.SupportsSystem(system) {
		return nil, fmt.Errorf("%s@%s: %w (%s; supported: %s)",
			def.Name, version, ErrUnsupportedSystem, system, strings.Join(def.Platforms, ", "))
	}
	dir := c.InstallDir(def.Name, version, system)

	fingerprint := def.InstallFingerprint(system, version)

	// Fast path: no lock needed to read a finished install.
	if m, ok := readMarker(dir); ok && m.InstallHash == fingerprint {
		return c.installedFromMarker(def, version, dir, m, artifact)
	}

	unlock, err := lockDir(dir)
	if err != nil {
		return nil, err
	}
	defer unlock()

	// Another process may have finished while we waited for the lock.
	if m, ok := readMarker(dir); ok && m.InstallHash == fingerprint {
		return c.installedFromMarker(def, version, dir, m, artifact)
	}
	return c.install(ctx, def, version, system, dir, fingerprint, artifact)
}

func (c *Client) installedFromMarker(def *Definition, version, dir string, m marker, artifact Artifact) (*Installed, error) {
	for _, d := range []struct{ algo, want, got string }{
		{algoSHA256, artifact.Sha256, m.Sha256},
		{algoSHA512, artifact.Sha512, m.Sha512},
	} {
		if d.want != "" && d.got != "" && !strings.EqualFold(d.want, d.got) {
			return nil, fmt.Errorf("%s@%s: %w: %s was installed from an artifact with %s %s but the lockfile expects %s; remove the directory to reinstall",
				def.Name, version, ErrChecksumMismatch, dir, d.algo, d.got, d.want)
		}
	}
	return c.installed(def, dir, m.URL, m.Sha256), nil
}

func (c *Client) install(ctx context.Context, def *Definition, version, system, dir, fingerprint string, artifact Artifact) (*Installed, error) {
	install := def.InstallFor(system, version)
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

	switch {
	case artifact.URL != "":
	case install.Lookup != nil:
		found, err := c.lookupArtifact(ctx, def, install.Lookup, vars)
		if err != nil {
			return nil, fmt.Errorf("%s@%s: %w", def.Name, version, err)
		}
		artifact.URL = found.URL
		if !artifact.HasDigest() {
			artifact.Sha256, artifact.Sha512 = found.Sha256, found.Sha512
		}
	default:
		artifact.URL, err = Render(install.URL, vars)
		if err != nil {
			return nil, err
		}
	}
	vars = vars.withURL(artifact.URL)
	fetchURL := def.ResolveURL(artifact.URL)

	// Download next to the install dir so that a rename into place stays on
	// one filesystem.
	downloadDir := filepath.Join(c.CacheDir, ".downloads", def.Name, version, system)
	archivePath := filepath.Join(downloadDir, vars.Filename)
	sums, err := c.download(ctx, fetchURL, archivePath)
	if err != nil {
		return nil, fmt.Errorf("%s@%s: %w", def.Name, version, err)
	}
	if err := sums.verify(artifact); err != nil {
		_ = os.Remove(archivePath)
		return nil, fmt.Errorf("%s@%s: %w: %s %w", def.Name, version, ErrChecksumMismatch, fetchURL, err)
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
	if err := makeBinariesExecutable(staging, def.BinDirs(), def.Binaries); err != nil {
		return nil, err
	}

	if err := os.Rename(staging, dir); err != nil {
		return nil, err
	}

	if err := c.runPostInstall(ctx, def, dir, vars); err != nil {
		// Leave the directory in place for debugging, but without a marker
		// so the next install retries.
		return nil, fmt.Errorf("%s@%s: %w", def.Name, version, err)
	}
	if missing := missingBinaries(dir, def.BinDirs(), def.Binaries); len(missing) > 0 {
		return nil, fmt.Errorf("%s@%s: installed, but %s not found in %s (bin: %s)",
			def.Name, version, strings.Join(missing, ", "), dir, strings.Join(def.BinDirs(), ", "))
	}

	m := marker{
		Name: def.Name, Version: version, System: system, URL: artifact.URL,
		Sha256: sums.sha256, Sha512: sums.sha512, InstallHash: fingerprint,
	}
	if err := writeMarker(dir, m); err != nil {
		return nil, err
	}
	return c.installed(def, dir, artifact.URL, sums.sha256), nil
}

// makeBinariesExecutable sets the executable bits on every file named in
// binaries that exists in one of binDirs. Archives built by tools that do
// not preserve modes (npm tarballs, for example) ship scripts without them.
func makeBinariesExecutable(root string, binDirs, binaries []string) error {
	for _, dir := range binDirs {
		for _, name := range binaries {
			p := filepath.Join(root, dir, name)
			info, err := os.Stat(p)
			if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0o111 {
				continue
			}
			if err := os.Chmod(p, info.Mode().Perm()|0o111); err != nil {
				return err
			}
		}
	}
	return nil
}

// missingBinaries returns the names in binaries that are not present in
// any of binDirs.
func missingBinaries(root string, binDirs, binaries []string) []string {
	var missing []string
	for _, name := range binaries {
		found := false
		for _, dir := range binDirs {
			if _, err := os.Stat(filepath.Join(root, dir, name)); err == nil {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, name)
		}
	}
	return missing
}

// IsInstalled reports whether version of def is fully installed on system
// from the definition's current install recipe.
func (c *Client) IsInstalled(def *Definition, version, system string) bool {
	m, ok := readMarker(c.InstallDir(def.Name, version, system))
	return ok && m.InstallHash == def.InstallFingerprint(system, version)
}

func (c *Client) installed(def *Definition, dir, url, sha string) *Installed {
	bins := make([]string, 0, len(def.BinDirs()))
	for _, b := range def.BinDirs() {
		bins = append(bins, filepath.Join(dir, b))
	}
	return &Installed{Dir: dir, URL: url, Sha256: sha, BinDirs: bins}
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

// fileDigests are the digests of a downloaded file in every algorithm a
// lockfile may record.
type fileDigests struct {
	sha256 string
	sha512 string
}

// verify checks the digests against those recorded for the artifact.
func (f fileDigests) verify(artifact Artifact) error {
	if artifact.Sha256 != "" && !strings.EqualFold(f.sha256, artifact.Sha256) {
		return fmt.Errorf("has sha256 %s, lockfile expects %s", f.sha256, artifact.Sha256)
	}
	if artifact.Sha512 != "" && !strings.EqualFold(f.sha512, artifact.Sha512) {
		return fmt.Errorf("has sha512 %s, lockfile expects %s", f.sha512, artifact.Sha512)
	}
	return nil
}

// download fetches rawURL to dest (unless dest already exists) and returns
// the digests of the file.
func (c *Client) download(ctx context.Context, rawURL, dest string) (fileDigests, error) {
	if sums, err := digestFile(dest); err == nil {
		return sums, nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fileDigests{}, err
	}

	body, err := c.open(ctx, rawURL, "")
	if err != nil {
		return fileDigests{}, err
	}
	defer body.Close()

	tmp, err := os.CreateTemp(filepath.Dir(dest), filepath.Base(dest)+".part-")
	if err != nil {
		return fileDigests{}, err
	}
	defer os.Remove(tmp.Name())

	h256, h512 := sha256.New(), sha512.New()
	if _, err := io.Copy(io.MultiWriter(tmp, h256, h512), body); err != nil {
		tmp.Close()
		return fileDigests{}, fmt.Errorf("downloading %s: %w", rawURL, err)
	}
	if err := tmp.Close(); err != nil {
		return fileDigests{}, err
	}
	if err := os.Rename(tmp.Name(), dest); err != nil {
		return fileDigests{}, err
	}
	return fileDigests{hex.EncodeToString(h256.Sum(nil)), hex.EncodeToString(h512.Sum(nil))}, nil
}

func digestFile(path string) (fileDigests, error) {
	f, err := os.Open(path)
	if err != nil {
		return fileDigests{}, err
	}
	defer f.Close()
	h256, h512 := sha256.New(), sha512.New()
	if _, err := io.Copy(io.MultiWriter(h256, h512), f); err != nil {
		return fileDigests{}, err
	}
	return fileDigests{hex.EncodeToString(h256.Sum(nil)), hex.EncodeToString(h512.Sum(nil))}, nil
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

		// /bin/sh rather than a PATH lookup: the hook must run even when
		// devbox itself was started with a minimal PATH.
		cmd := exec.CommandContext(ctx, "/bin/sh", "-c", script)
		cmd.Dir = dir
		cmd.Stdout = output
		cmd.Stderr = output
		cmd.Env = append(os.Environ(),
			"DEVBOX_PKG_INSTALL_DIR="+dir,
			"DEVBOX_PKG_VERSION="+vars.Version,
			"DEVBOX_PKG_SYSTEM="+vars.System,
			// Hooks may use basic utilities (ln, chmod, ...) even when
			// devbox was started with a minimal PATH.
			"PATH="+hookPath(os.Getenv("PATH")),
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

// hookPath returns path with the standard system directories appended.
func hookPath(path string) string {
	if path == "" {
		return "/usr/bin:/bin"
	}
	return path + ":/usr/bin:/bin"
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
