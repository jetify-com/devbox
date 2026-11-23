// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSystem = "x86_64-linux"

// fakeVendor is an httptest server that behaves like a vendor download
// site: a JSON index of versions with checksums, tarballs, a bare binary,
// a SHASUMS file and a GitHub releases endpoint.
type fakeVendor struct {
	srv     *httptest.Server
	tarball []byte
	tarSHA  string
	binary  []byte
	binSHA  string

	mu   sync.Mutex
	hits map[string]int
}

func (vendor *fakeVendor) hit(key string) {
	vendor.mu.Lock()
	defer vendor.mu.Unlock()
	vendor.hits[key]++
}

func (vendor *fakeVendor) hitCount(key string) int {
	vendor.mu.Lock()
	defer vendor.mu.Unlock()
	return vendor.hits[key]
}

func newFakeVendor(t *testing.T) *fakeVendor {
	t.Helper()
	vendor := &fakeVendor{hits: map[string]int{}}
	vendor.tarball = makeTarGz(t, map[string]string{
		"hello-1.2.0/bin/hello": "#!/bin/sh\necho hello 1.2.0\n",
		"hello-1.2.0/README":    "readme\n",
	})
	vendor.tarSHA = sha256Hex(vendor.tarball)
	vendor.binary = []byte("#!/bin/sh\necho hello binary\n")
	vendor.binSHA = sha256Hex(vendor.binary)

	mux := http.NewServeMux()
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, _ *http.Request) {
		vendor.hit("/index.json")
		fmt.Fprintf(w, `[
		  {"version": "v1.2.0", "stable": true, "files": [
		    {"filename": "hello-1.2.0-linux-amd64.tar.gz", "sha256": %q},
		    {"filename": "hello-1.2.0-darwin-arm64.tar.gz", "sha256": "%s"}
		  ]},
		  {"version": "v1.1.0", "stable": true, "files": []},
		  {"version": "v1.3.0-rc1", "stable": false, "files": []}
		]`, vendor.tarSHA, strings.Repeat("ab", 32))
	})
	mux.HandleFunc("/dl/", func(w http.ResponseWriter, r *http.Request) {
		vendor.hit(r.URL.Path)
		switch {
		case strings.HasSuffix(r.URL.Path, ".tar.gz"):
			_, _ = w.Write(vendor.tarball)
		case strings.HasSuffix(r.URL.Path, "/hello"):
			_, _ = w.Write(vendor.binary)
		case strings.HasSuffix(r.URL.Path, "SHASUMS256.txt"):
			fmt.Fprintf(w, "%s  hello-1.2.0-linux-amd64.tar.gz\n%s  *hello\n", vendor.tarSHA, vendor.binSHA)
		case strings.HasSuffix(r.URL.Path, ".sha256"):
			fmt.Fprintf(w, "%s\n", vendor.binSHA)
		default:
			http.NotFound(w, r)
		}
	})
	mux.HandleFunc("/repos/acme/hello/releases", func(w http.ResponseWriter, r *http.Request) {
		vendor.hit("github")
		if auth := r.Header.Get("Authorization"); auth != "" {
			w.Header().Set("X-Seen-Auth", auth)
		}
		_, _ = io.WriteString(w, `[
		  {"tag_name": "v2.0.0-beta.1", "draft": false, "prerelease": true},
		  {"tag_name": "v1.5.0", "draft": true, "prerelease": false},
		  {"tag_name": "v1.4.2", "draft": false, "prerelease": false},
		  {"tag_name": "v1.4.1", "draft": false, "prerelease": false},
		  {"tag_name": "nightly", "draft": false, "prerelease": false}
		]`)
	})
	vendor.srv = httptest.NewServer(mux)
	t.Cleanup(vendor.srv.Close)
	return vendor
}

func (vendor *fakeVendor) client(t *testing.T) *Client {
	t.Helper()
	cli := NewClient()
	cli.HTTPClient = vendor.srv.Client()
	cli.CacheDir = filepath.Join(t.TempDir(), "pkgs")
	cli.GitHubAPIURL = vendor.srv.URL
	cli.GitHubToken = ""
	return cli
}

func (vendor *fakeVendor) archiveDef(t *testing.T) *Definition {
	t.Helper()
	def, err := Parse([]byte(fmt.Sprintf(`{
	  "schema": 1,
	  "name": "hello",
	  "platforms": ["x86_64-linux", "aarch64-darwin"],
	  "versions": {"type": "http_json", "url": %[1]q, "select": ".[] | select(.stable) | .version | ltrimstr(\"v\")"},
	  "install": {
	    "type": "archive",
	    "url": "%[2]s/dl/hello-{{.Version}}-{{.OS}}-{{.Arch}}.tar.gz",
	    "checksum": {"type": "http_json", "url": %[1]q, "select": ".[].files[] | select(.filename == \"{{.Filename}}\") | .sha256"},
	    "strip_components": 1
	  },
	  "binaries": ["hello"],
	  "hooks": {"post_install": ["echo installed {{.Version}} > $DEVBOX_PKG_INSTALL_DIR/hook.txt"]}
	}`, vendor.srv.URL+"/index.json", vendor.srv.URL)))
	require.NoError(t, err)
	return def
}

func (vendor *fakeVendor) binaryDef(t *testing.T, checksum string) *Definition {
	t.Helper()
	def, err := Parse([]byte(fmt.Sprintf(`{
	  "schema": 1,
	  "name": "hello-bin",
	  "platforms": ["x86_64-linux"],
	  "versions": {"type": "static", "versions": ["1.2.0"]},
	  "install": {"type": "binary", "url": "%s/dl/{{.Version}}/hello", "bin_name": "hi", %s}
	}`, vendor.srv.URL, checksum)))
	require.NoError(t, err)
	return def
}

func TestResolveAndLock(t *testing.T) {
	vendor := newFakeVendor(t)
	cli := vendor.client(t)
	def := vendor.archiveDef(t)

	versions, err := cli.ListVersions(t.Context(), def)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"1.2.0", "1.1.0"}, versions)

	resolved, err := cli.Resolve(t.Context(), def, "1")
	require.NoError(t, err)
	assert.Equal(t, "1.2.0", resolved.Version)
	assert.Equal(t, map[string]Artifact{
		"x86_64-linux":   {URL: vendor.srv.URL + "/dl/hello-1.2.0-linux-amd64.tar.gz", Sha256: vendor.tarSHA},
		"aarch64-darwin": {URL: vendor.srv.URL + "/dl/hello-1.2.0-darwin-arm64.tar.gz", Sha256: strings.Repeat("ab", 32)},
	}, resolved.Artifacts)

	_, err = cli.Resolve(t.Context(), def, "9")
	require.ErrorIs(t, err, ErrVersionNotFound)

	// A version with no checksum entry in the index is a lock error, not a
	// silently empty digest.
	_, err = cli.Lock(t.Context(), def, "1.1.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "matched nothing")
}

func TestInstallArchive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh is required")
	}
	vendor := newFakeVendor(t)
	cli := vendor.client(t)
	def := vendor.archiveDef(t)
	art := Artifact{URL: vendor.srv.URL + "/dl/hello-1.2.0-linux-amd64.tar.gz", Sha256: vendor.tarSHA}

	assert.False(t, cli.IsInstalled(def, "1.2.0", testSystem, art))
	got, err := cli.Install(t.Context(), def, "1.2.0", testSystem, art)
	require.NoError(t, err)
	assert.Equal(t, installDir(t, cli, def, "1.2.0", art), got.Dir)
	assert.Equal(t, vendor.tarSHA, got.Sha256)
	assert.Equal(t, art.URL, got.URL)
	assert.Equal(t, []string{filepath.Join(got.Dir, "bin")}, got.BinDirs)
	assert.True(t, cli.IsInstalled(def, "1.2.0", testSystem, art))

	// strip_components removed the hello-1.2.0/ directory.
	content, err := os.ReadFile(filepath.Join(got.Dir, "bin", "hello"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "echo hello 1.2.0")
	_, err = os.Stat(filepath.Join(got.Dir, "README"))
	require.NoError(t, err)

	// The post_install hook ran with the install dir in its environment.
	hook, err := os.ReadFile(filepath.Join(got.Dir, "hook.txt"))
	require.NoError(t, err)
	assert.Equal(t, "installed 1.2.0\n", string(hook))

	// The download is removed after a successful install.
	_, err = os.Stat(cli.downloadDir(got.Dir, art.URL))
	assert.True(t, os.IsNotExist(err))

	// A second install is served from disk without touching the network.
	hits := vendor.hits["/dl/hello-1.2.0-linux-amd64.tar.gz"]
	again, err := cli.Install(t.Context(), def, "1.2.0", testSystem, art)
	require.NoError(t, err)
	assert.Equal(t, got, again)
	assert.Equal(t, hits, vendor.hits["/dl/hello-1.2.0-linux-amd64.tar.gz"])

	// Installing with a URL left empty renders it from the definition.
	cli2 := vendor.client(t)
	got2, err := cli2.Install(t.Context(), def, "1.2.0", testSystem, Artifact{Sha256: vendor.tarSHA})
	require.NoError(t, err)
	assert.Equal(t, vendor.tarSHA, got2.Sha256)
}

func TestInstallChecksumMismatch(t *testing.T) {
	vendor := newFakeVendor(t)
	cli := vendor.client(t)
	def := vendor.archiveDef(t)
	art := Artifact{URL: vendor.srv.URL + "/dl/hello-1.2.0-linux-amd64.tar.gz", Sha256: strings.Repeat("00", 32)}

	_, err := cli.Install(t.Context(), def, "1.2.0", testSystem, art)
	require.ErrorIs(t, err, ErrChecksumMismatch)
	assert.False(t, cli.IsInstalled(def, "1.2.0", testSystem, art))
	_, statErr := os.Stat(installDir(t, cli, def, "1.2.0", art))
	assert.True(t, os.IsNotExist(statErr), "nothing should be installed")
	_, statErr = os.Stat(cli.downloadDir(installDir(t, cli, def, "1.2.0", art), art.URL))
	assert.True(t, os.IsNotExist(statErr), "the rejected download is removed")

	// An existing install from a different artifact is also rejected.
	_, err = cli.Install(t.Context(), def, "1.2.0", testSystem, Artifact{Sha256: vendor.tarSHA})
	require.NoError(t, err)
	_, err = cli.Install(t.Context(), def, "1.2.0", testSystem, art)
	require.ErrorIs(t, err, ErrChecksumMismatch)
}

func TestInstallUnsupportedSystem(t *testing.T) {
	vendor := newFakeVendor(t)
	cli := vendor.client(t)
	def := vendor.archiveDef(t)
	_, err := cli.Install(t.Context(), def, "1.2.0", "aarch64-linux", Artifact{})
	require.ErrorIs(t, err, ErrUnsupportedSystem)
}

func TestInstallBinaryTrustOnFirstUse(t *testing.T) {
	vendor := newFakeVendor(t)
	cli := vendor.client(t)
	def := vendor.binaryDef(t, `"checksum": {"type": "none"}`)

	artifacts, err := cli.Lock(t.Context(), def, "1.2.0")
	require.NoError(t, err)
	assert.Equal(t, Artifact{URL: vendor.srv.URL + "/dl/1.2.0/hello"}, artifacts[testSystem], "no digest without a checksum source")

	got, err := cli.Install(t.Context(), def, "1.2.0", testSystem, artifacts[testSystem])
	require.NoError(t, err)
	assert.Equal(t, vendor.binSHA, got.Sha256, "the observed digest is reported so the caller can lock it")

	bin := filepath.Join(got.Dir, "bin", "hi")
	info, err := os.Stat(bin)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&0o111, "binary must be executable")
	content, err := os.ReadFile(bin)
	require.NoError(t, err)
	assert.Equal(t, vendor.binary, content)
}

func TestChecksumSHA256URL(t *testing.T) {
	vendor := newFakeVendor(t)
	cli := vendor.client(t)

	// SHASUMS-style list matched by filename (with a "*" binary marker).
	def := vendor.binaryDef(t, fmt.Sprintf(`"checksum": {"type": "sha256_url", "url": "%s/dl/SHASUMS256.txt"}`, vendor.srv.URL))
	artifacts, err := cli.Lock(t.Context(), def, "1.2.0")
	require.NoError(t, err)
	assert.Equal(t, vendor.binSHA, artifacts[testSystem].Sha256)

	// Bare digest at <url>.sha256.
	def = vendor.binaryDef(t, `"checksum": {"type": "sha256_url", "url": "{{.URL}}.sha256"}`)
	artifacts, err = cli.Lock(t.Context(), def, "1.2.0")
	require.NoError(t, err)
	assert.Equal(t, vendor.binSHA, artifacts[testSystem].Sha256)

	// Missing checksum file is an error.
	def = vendor.binaryDef(t, fmt.Sprintf(`"checksum": {"type": "sha256_url", "url": "%s/dl/missing.txt"}`, vendor.srv.URL))
	_, err = cli.Lock(t.Context(), def, "1.2.0")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestParseSHA256Body(t *testing.T) {
	digest := strings.Repeat("0a", 32)
	got, err := parseSHA256Body(digest+"\n", "x", "src")
	require.NoError(t, err)
	assert.Equal(t, digest, got)

	got, err = parseSHA256Body("sha256:"+strings.ToUpper(digest), "x", "src")
	require.NoError(t, err)
	assert.Equal(t, digest, got)

	body := strings.Repeat("11", 32) + "  other.tar.gz\n" + digest + "  dist/x.tar.gz\n"
	got, err = parseSHA256Body(body, "x.tar.gz", "src")
	require.NoError(t, err)
	assert.Equal(t, digest, got)

	_, err = parseSHA256Body(body, "nope.tar.gz", "src")
	require.Error(t, err)

	_, err = parseSHA256Body("nothex", "x", "src")
	require.Error(t, err)
}

func TestListVersionsGitHub(t *testing.T) {
	vendor := newFakeVendor(t)
	cli := vendor.client(t)
	def, err := Parse([]byte(`{
	  "schema": 1, "name": "hello", "platforms": ["x86_64-linux"],
	  "versions": {"type": "github_releases", "repo": "acme/hello"},
	  "install": {"type": "archive", "url": "u"}
	}`))
	require.NoError(t, err)

	versions, err := cli.ListVersions(t.Context(), def)
	require.NoError(t, err)
	assert.Equal(t, []string{"1.4.2", "1.4.1"}, versions, "drafts, pre-releases and tags without the prefix are skipped")

	version, err := cli.ResolveVersion(t.Context(), def, "latest")
	require.NoError(t, err)
	assert.Equal(t, "1.4.2", version)

	// An empty strip_prefix keeps every tag.
	empty := ""
	def.Versions.StripPrefix = &empty
	versions, err = cli.ListVersions(t.Context(), def)
	require.NoError(t, err)
	assert.Equal(t, []string{"v1.4.2", "v1.4.1", "nightly"}, versions)
}

func TestNotImplementedStrategies(t *testing.T) {
	vendor := newFakeVendor(t)
	cli := vendor.client(t)

	def, err := Parse([]byte(`{
	  "schema": 1, "name": "hello", "platforms": ["x86_64-linux"],
	  "versions": {"type": "script", "run": "echo 1"},
	  "install": {"type": "script", "run": "true"}
	}`))
	require.NoError(t, err)
	assert.True(t, def.UsesScripts())

	_, err = cli.ListVersions(t.Context(), def)
	require.ErrorIs(t, err, ErrNotImplemented)
	_, err = cli.Lock(t.Context(), def, "1")
	require.ErrorIs(t, err, ErrNotImplemented)
	_, err = cli.Install(t.Context(), def, "1", testSystem, Artifact{})
	require.ErrorIs(t, err, ErrNotImplemented)
}

func TestLoadLocal(t *testing.T) {
	cli := NewClient()
	cli.CacheDir = t.TempDir()
	projectDir := t.TempDir()
	path := filepath.Join(projectDir, "defs", "hello.pkg.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(minimalDef), 0o644))

	for _, source := range []string{"./defs/hello.pkg.json", path, "file://" + path} {
		loaded, err := cli.Load(t.Context(), Ref{Source: source}, projectDir, "ignored-for-local")
		require.NoError(t, err, source)
		assert.Equal(t, "hello", loaded.Definition.Name)
		assert.Equal(t, sha256Hex([]byte(minimalDef)), loaded.Hash)
		assert.Equal(t, filepath.Dir(path), loaded.Dir)
	}

	_, err := cli.Load(t.Context(), Ref{Source: "./missing.pkg.json"}, projectDir, "")
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestLoadRemote(t *testing.T) {
	content := []byte(minimalDef)
	served := content
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hello.pkg.json" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(served)
	}))
	t.Cleanup(srv.Close)

	cli := NewClient()
	cli.HTTPClient = srv.Client()
	cli.CacheDir = t.TempDir()
	ref := Ref{Source: srv.URL + "/hello.pkg.json", Version: "1"}

	loaded, err := cli.Load(t.Context(), ref, "", "")
	require.NoError(t, err)
	assert.Equal(t, "hello", loaded.Definition.Name)
	assert.Equal(t, sha256Hex(content), loaded.Hash)
	assert.Empty(t, loaded.Dir)

	// Pinned to the hash we just saw: served from cache even if the server
	// now returns something else.
	served = []byte(strings.Replace(minimalDef, `"hello"`, `"changed"`, 1))
	pinned, err := cli.Load(t.Context(), ref, "", loaded.Hash)
	require.NoError(t, err)
	assert.Equal(t, "hello", pinned.Definition.Name)

	// Pinned to a hash that is not cached and does not match: rejected.
	_, err = cli.Load(t.Context(), ref, "", strings.Repeat("ff", 32))
	require.ErrorIs(t, err, ErrDefinitionChanged)

	// Unpinned picks up the new content.
	fresh, err := cli.Load(t.Context(), ref, "", "")
	require.NoError(t, err)
	assert.Equal(t, "changed", fresh.Definition.Name)

	_, err = cli.Load(t.Context(), Ref{Source: srv.URL + "/missing.pkg.json"}, "", "")
	require.ErrorIs(t, err, ErrNotFound)

	// Plain http is refused even when the Ref is built in code.
	_, err = cli.Load(t.Context(), Ref{Source: "http://example.com/hello.pkg.json"}, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must use https://")
}

func TestRemoteDefinitionRejectsFileURLs(t *testing.T) {
	var served string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, served)
	}))
	t.Cleanup(srv.Close)
	cli := NewClient()
	cli.HTTPClient = srv.Client()
	cli.CacheDir = t.TempDir()
	ref := Ref{Source: srv.URL + "/x.pkg.json"}

	secret := filepath.Join(t.TempDir(), "secret.json")
	require.NoError(t, os.WriteFile(secret, []byte(`["1.0.0"]`), 0o644))
	const tmpl = `{
	  "schema": 1, "name": "x", "platforms": ["x86_64-linux"],
	  "versions": {"type": "http_json", "url": %q, "select": ".[]"},
	  "install": {"type": "archive", "url": %q, "checksum": {"type": "sha256_url", "url": %q},
	    "overrides": {"x86_64-linux": {"url": %q}}}
	}`
	good := "https://example.com/x"
	tests := map[string][4]string{
		"versions":         {"file://" + secret, good, good, good},
		"relative":         {"file://secret.json", good, good, good},
		"install":          {good, "FILE:///etc/passwd", good, good},
		"checksum":         {good, good, "file://" + secret, good},
		"install override": {good, good, good, "file:///etc/passwd"},
	}
	for name, urls := range tests {
		t.Run(name, func(t *testing.T) {
			served = fmt.Sprintf(tmpl, urls[0], urls[1], urls[2], urls[3])
			_, err := cli.Load(t.Context(), ref, "", "")
			require.ErrorIs(t, err, ErrFileURLInRemoteDefinition)
		})
	}

	// A file:// URL assembled by a template is caught after rendering.
	served = fmt.Sprintf(`{
	  "schema": 1, "name": "x", "platforms": ["x86_64-linux"], "os_map": {"linux": "file"},
	  "versions": {"type": "static", "versions": ["1.0.0"]},
	  "install": {"type": "binary", "url": "{{.OS}}://%s",
	    "checksum": {"type": "sha256_url", "url": "{{.OS}}://%s"}}
	}`, secret, secret)
	loaded, err := cli.Load(t.Context(), ref, "", "")
	require.NoError(t, err)
	_, err = cli.Lock(t.Context(), loaded.Definition, "1.0.0")
	require.ErrorIs(t, err, ErrFileURLInRemoteDefinition)
	_, err = cli.Install(t.Context(), loaded.Definition, "1.0.0", testSystem, Artifact{})
	require.ErrorIs(t, err, ErrFileURLInRemoteDefinition)
	_, err = cli.Install(t.Context(), loaded.Definition, "1.0.0", testSystem, Artifact{URL: "file://" + secret})
	require.ErrorIs(t, err, ErrFileURLInRemoteDefinition)

	// Local definitions keep file:// support.
	dir := t.TempDir()
	path := filepath.Join(dir, "x.pkg.json")
	require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf(tmpl, "file://"+secret, good, good, good)), 0o644))
	local, err := cli.Load(t.Context(), Ref{Source: path}, dir, "")
	require.NoError(t, err)
	versions, err := cli.ListVersions(t.Context(), local.Definition)
	require.NoError(t, err)
	assert.Equal(t, []string{"1.0.0"}, versions)
}

func TestListVersionsDropsInvalidVersions(t *testing.T) {
	cli := NewClient()
	cli.CacheDir = t.TempDir()
	index := filepath.Join(t.TempDir(), "index.json")
	require.NoError(t, os.WriteFile(index, []byte(`["1.0.0", "99+/../../../tmp/evil", "1.24.0+$(touch x)", " 2.0", "-1", ".1", "", "1.1.0+build_2"]`), 0o644))
	def, err := Parse([]byte(fmt.Sprintf(`{
	  "schema": 1, "name": "x", "platforms": ["x86_64-linux"],
	  "versions": {"type": "http_json", "url": "file://%s", "select": ".[]"},
	  "install": {"type": "archive", "url": "https://example.com/x"}
	}`, index)))
	require.NoError(t, err)

	versions, err := cli.ListVersions(t.Context(), def)
	require.NoError(t, err)
	assert.Equal(t, []string{"1.0.0", "1.1.0+build_2"}, versions)
	latest, err := cli.ResolveVersion(t.Context(), def, Latest)
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", latest, "never the escaping version; 1.1.0+build_2 has letters, so it is a pre-release")

	def.Versions = VersionSource{Type: VersionsStatic, Versions: []string{"1.0.0", "../evil"}}
	versions, err = cli.ListVersions(t.Context(), def)
	require.NoError(t, err)
	assert.Equal(t, []string{"1.0.0"}, versions)

	// An invalid version never reaches the filesystem.
	for _, version := range []string{"99+/../../../tmp/evil", "../x", "1 2"} {
		_, err = cli.Install(t.Context(), def, version, testSystem, Artifact{})
		require.Error(t, err, version)
		assert.Contains(t, err.Error(), "invalid version", version)
		_, err = cli.InstallDir(def, version, testSystem, Artifact{})
		require.Error(t, err, version)
	}
	entries, err := os.ReadDir(cli.CacheDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestInstallDirKey(t *testing.T) {
	vendor := newFakeVendor(t)
	cli := vendor.client(t)
	def := vendor.archiveDef(t)
	url := vendor.srv.URL + "/dl/hello-1.2.0-linux-amd64.tar.gz"
	art := Artifact{URL: url, Sha256: vendor.tarSHA}

	dir := installDir(t, cli, def, "1.2.0", art)
	assert.Equal(t, filepath.Join(cli.CacheDir, "hello", "1.2.0"), filepath.Dir(dir))
	assert.Regexp(t, `^x86_64-linux-[0-9a-f]{12}$`, filepath.Base(dir))

	// An empty URL is rendered from the definition, so it is the same
	// install; digest case does not matter.
	assert.Equal(t, dir, installDir(t, cli, def, "1.2.0", Artifact{Sha256: vendor.tarSHA}))
	assert.Equal(t, dir, installDir(t, cli, def, "1.2.0", Artifact{URL: url, Sha256: strings.ToUpper(vendor.tarSHA)}))

	// The digest is not part of the key, so recording it (trust on first
	// use) does not move the install.
	assert.Equal(t, dir, installDir(t, cli, def, "1.2.0", Artifact{URL: url}))

	// Anything else that changes the installed bytes changes the key.
	others := map[string]string{
		"url":        installDir(t, cli, def, "1.2.0", Artifact{URL: url + "?mirror", Sha256: vendor.tarSHA}),
		"version":    filepath.Base(installDir(t, cli, def, "1.2.1", art)),
		"definition": "",
	}
	other := vendor.archiveDef(t)
	other.Hooks.PostInstall = []string{"true"}
	others["definition"] = installDir(t, cli, other, "1.2.0", art)
	for name, got := range others {
		assert.NotEqual(t, dir, got, name)
	}
	assert.NotEqual(t, def.InstallFingerprint(testSystem), other.InstallFingerprint(testSystem))
}

func TestInstallsAreImmutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh is required")
	}
	vendor := newFakeVendor(t)
	cli := vendor.client(t)
	art := Artifact{Sha256: vendor.tarSHA}

	// Two definitions with the same name and version but different hooks
	// get their own directories and never replace each other.
	def1 := vendor.archiveDef(t)
	def2 := vendor.archiveDef(t)
	def2.Hooks.PostInstall = []string{"echo two > hook.txt"}
	got1, err := cli.Install(t.Context(), def1, "1.2.0", testSystem, art)
	require.NoError(t, err)
	got2, err := cli.Install(t.Context(), def2, "1.2.0", testSystem, art)
	require.NoError(t, err)
	assert.NotEqual(t, got1.Dir, got2.Dir)
	for dir, want := range map[string]string{got1.Dir: "installed 1.2.0\n", got2.Dir: "two\n"} {
		hook, err := os.ReadFile(filepath.Join(dir, "hook.txt"))
		require.NoError(t, err)
		assert.Equal(t, want, string(hook))
	}
	again, err := cli.Install(t.Context(), def1, "1.2.0", testSystem, art)
	require.NoError(t, err)
	assert.Equal(t, got1, again)
	assert.Equal(t, 2, vendor.hitCount("/dl/hello-1.2.0-linux-amd64.tar.gz"))

	// Each install downloads into its own directory, so concurrent
	// installs of the same URL under different keys never remove each
	// other's download.
	assert.NotEqual(t, cli.downloadDir(got1.Dir, got1.URL), cli.downloadDir(got2.Dir, got2.URL))
	cli2 := vendor.client(t)
	defs := []*Definition{def1, def2, vendor.archiveDef(t), vendor.archiveDef(t)}
	defs[2].Hooks.PostInstall = []string{"true"}
	defs[3].Hooks.PostInstall = []string{"true", "true"}
	results := make(chan error, len(defs))
	for _, def := range defs {
		go func() {
			_, err := cli2.Install(t.Context(), def, "1.2.0", testSystem, art)
			results <- err
		}()
	}
	for range defs {
		require.NoError(t, <-results)
	}
	entries, err := os.ReadDir(filepath.Join(cli2.CacheDir, ".downloads"))
	require.NoError(t, err)
	assert.Empty(t, entries)

	// The lock file sits next to the install.
	_, err = os.Stat(got1.Dir + ".lock")
	require.NoError(t, err)
}

func TestInstallTrustOnFirstUseRequiresMatchingURL(t *testing.T) {
	vendor := newFakeVendor(t)
	cli := vendor.client(t)
	def := vendor.binaryDef(t, `"checksum": {"type": "none"}`)
	art := Artifact{URL: vendor.srv.URL + "/dl/1.2.0/hello"}

	got, err := cli.Install(t.Context(), def, "1.2.0", testSystem, art)
	require.NoError(t, err)
	assert.Equal(t, art.URL, got.URL)
	assert.Equal(t, vendor.binSHA, got.Sha256)

	// Recording the digest keeps the install where it is.
	locked := Artifact{URL: art.URL, Sha256: got.Sha256}
	assert.Equal(t, got.Dir, installDir(t, cli, def, "1.2.0", locked))

	// A marker from another URL (another definition's install) is never
	// accepted, so its digest is never reported for this URL.
	m, ok := readMarker(got.Dir)
	require.True(t, ok)
	m.URL = "https://elsewhere.example.com/hello"
	m.Sha256 = strings.Repeat("ee", 32)
	require.NoError(t, writeMarker(got.Dir, m))
	assert.False(t, cli.IsInstalled(def, "1.2.0", testSystem, art))
	_, err = cli.Install(t.Context(), def, "1.2.0", testSystem, art)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "was installed from https://elsewhere.example.com/hello")
	_, err = os.Stat(filepath.Join(got.Dir, "bin", "hi"))
	require.NoError(t, err, "the completed install is left alone")
}

func TestGetErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, "slow down")
	}))
	t.Cleanup(srv.Close)
	cli := NewClient()
	cli.HTTPClient = srv.Client()

	_, err := cli.get(t.Context(), srv.URL+"/x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "429")
	assert.Contains(t, err.Error(), "slow down")
}

func makeTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	for name, content := range files {
		mode := int64(0o644)
		if strings.Contains(name, "/bin/") {
			mode = 0o755
		}
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: int64(len(content))}))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())
	return buf.Bytes()
}

func installDir(t *testing.T, cli *Client, def *Definition, version string, art Artifact) string {
	t.Helper()
	dir, err := cli.InstallDir(def, version, testSystem, art)
	require.NoError(t, err)
	return dir
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func TestGitHubTokenOnlyForGitHubAPI(t *testing.T) {
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.URL.Path+"|"+r.Header.Get("Authorization"))
		if strings.HasPrefix(r.URL.Path, "/repos/") {
			_, _ = io.WriteString(w, `[{"tag_name": "v1.0.0"}]`)
			return
		}
		_, _ = io.WriteString(w, `["1.0.0"]`)
	}))
	t.Cleanup(srv.Close)

	cli := NewClient()
	cli.HTTPClient = srv.Client()
	cli.CacheDir = t.TempDir()
	cli.GitHubAPIURL = srv.URL
	cli.GitHubToken = "secret"

	// A definition's own URLs (even on the same host as the API) never get
	// the token.
	def, err := Parse([]byte(fmt.Sprintf(`{
	  "schema": 1, "name": "x", "platforms": ["x86_64-linux"],
	  "versions": {"type": "http_json", "url": "%s/index.json", "select": ".[]"},
	  "install": {"type": "archive", "url": "u"}
	}`, srv.URL)))
	require.NoError(t, err)
	_, err = cli.ListVersions(t.Context(), def)
	require.NoError(t, err)

	def.Versions = VersionSource{Type: VersionsGitHubReleases, Repo: "acme/x"}
	_, err = cli.ListVersions(t.Context(), def)
	require.NoError(t, err)

	assert.Equal(t, []string{"/index.json|", "/repos/acme/x/releases|Bearer secret"}, seen)
}

func TestJQNumbersKeepTheirDigits(t *testing.T) {
	code, err := compileJQ(".[]")
	require.NoError(t, err)
	var input any
	require.NoError(t, json.Unmarshal([]byte(`[20240101, 1000000, 1.10, "1.2", true, null, [7, "8"]]`), &input))
	got, err := runJQ(context.Background(), code, ".[]", input)
	require.NoError(t, err)
	assert.Equal(t, []string{"20240101", "1000000", "1.1", "1.2", "7", "8"}, got)
}

func TestInstallConcurrent(t *testing.T) {
	vendor := newFakeVendor(t)
	cli := vendor.client(t)
	def := vendor.archiveDef(t)
	art := Artifact{Sha256: vendor.tarSHA}

	const workers = 4
	results := make(chan error, workers)
	for range workers {
		go func() {
			_, err := cli.Install(t.Context(), def, "1.2.0", testSystem, art)
			results <- err
		}()
	}
	for range workers {
		require.NoError(t, <-results)
	}
	assert.True(t, cli.IsInstalled(def, "1.2.0", testSystem, art))
	content, err := os.ReadFile(filepath.Join(installDir(t, cli, def, "1.2.0", art), "bin", "hello"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "echo hello 1.2.0")
	assert.Equal(t, 1, vendor.hitCount("/dl/hello-1.2.0-linux-amd64.tar.gz"), "the archive is downloaded once")
}

func TestInstallHookFailureIncludesOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh is required")
	}
	vendor := newFakeVendor(t)
	cli := vendor.client(t)
	def := vendor.archiveDef(t)
	def.Hooks.PostInstall = []string{"echo something went wrong >&2; exit 3"}

	art := Artifact{Sha256: vendor.tarSHA}
	_, err := cli.Install(t.Context(), def, "1.2.0", testSystem, art)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exit status 3")
	assert.Contains(t, err.Error(), "something went wrong")
	assert.False(t, cli.IsInstalled(def, "1.2.0", testSystem, art), "no marker after a failed hook")
	entries, err := os.ReadDir(filepath.Join(cli.CacheDir, ".downloads"))
	require.NoError(t, err)
	assert.Empty(t, entries, "the download is removed after a failed hook")

	info, err := os.Stat(installDir(t, cli, def, "1.2.0", art))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o755), info.Mode().Perm(), "install dir is world readable")
}

func TestInstallIgnoresArchiveMarker(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh is required")
	}
	vendor := newFakeVendor(t)
	// The archive ships its own marker claiming a finished install.
	vendor.tarball = makeTarGz(t, map[string]string{
		"hello-1.2.0/bin/hello":     "#!/bin/sh\necho hello 1.2.0\n",
		"hello-1.2.0/" + markerFile: `{}`,
	})
	vendor.tarSHA = sha256Hex(vendor.tarball)
	cli := vendor.client(t)
	def := vendor.archiveDef(t)
	def.Hooks.PostInstall = []string{"exit 3"}

	art := Artifact{Sha256: vendor.tarSHA}
	_, err := cli.Install(t.Context(), def, "1.2.0", testSystem, art)
	require.Error(t, err)
	dir := installDir(t, cli, def, "1.2.0", art)
	_, err = os.Stat(filepath.Join(dir, markerFile))
	assert.ErrorIs(t, err, os.ErrNotExist, "the archive's marker is removed")
	_, err = cli.Install(t.Context(), def, "1.2.0", testSystem, art)
	require.Error(t, err, "the failed install is retried, not accepted")
}

func TestDownloadDirIsPerInstall(t *testing.T) {
	cli := NewClient()
	cli.CacheDir = t.TempDir()
	url := "https://example.com/tool.tar.gz"
	// Two versions of one definition with a fixed URL share an install
	// key but not an install directory, so they must not share a download.
	a := filepath.Join(cli.CacheDir, "tool", "1.0.0", "x86_64-linux-abc")
	b := filepath.Join(cli.CacheDir, "tool", "2.0.0", "x86_64-linux-abc")
	assert.NotEqual(t, cli.downloadDir(a, url), cli.downloadDir(b, url))
}

func TestRunJQStopsOnCancel(t *testing.T) {
	code, err := compileJQ("repeat(1)")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = runJQ(ctx, code, "repeat(1)", nil)
	require.Error(t, err)

	_, err = runJQ(t.Context(), code, "repeat(1)", nil)
	require.ErrorContains(t, err, "more than")
}
