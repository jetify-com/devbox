// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
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

	assert.False(t, cli.IsInstalled(def, "1.2.0", testSystem))
	got, err := cli.Install(t.Context(), def, "1.2.0", testSystem, art)
	require.NoError(t, err)
	assert.Equal(t, cli.InstallDir("hello", "1.2.0", testSystem), got.Dir)
	assert.Equal(t, vendor.tarSHA, got.Sha256)
	assert.Equal(t, []string{filepath.Join(got.Dir, "bin")}, got.BinDirs)
	assert.True(t, cli.IsInstalled(def, "1.2.0", testSystem))

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
	_, err = os.Stat(filepath.Join(cli.CacheDir, ".downloads", "hello", "1.2.0", testSystem, "hello-1.2.0-linux-amd64.tar.gz"))
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
	assert.False(t, cli.IsInstalled(def, "1.2.0", testSystem))
	_, statErr := os.Stat(cli.InstallDir("hello", "1.2.0", testSystem))
	assert.True(t, os.IsNotExist(statErr), "nothing should be installed")

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

	// A missing checksum file means there is no artifact for any system.
	def = vendor.binaryDef(t, fmt.Sprintf(`"checksum": {"type": "sha256_url", "url": "%s/dl/missing.txt"}`, vendor.srv.URL))
	_, err = cli.Lock(t.Context(), def, "1.2.0")
	require.ErrorIs(t, err, ErrNoArtifact)
}

func TestParseChecksumFile(t *testing.T) {
	hexDigest := strings.Repeat("0a", 32)
	want := digest{algo: algoSHA256, hex: hexDigest}
	got, err := parseChecksumFile(hexDigest+"\n", "x", "src", algoSHA256)
	require.NoError(t, err)
	assert.Equal(t, want, got)

	got, err = parseChecksumFile("sha256:"+strings.ToUpper(hexDigest), "x", "src", algoSHA256)
	require.NoError(t, err)
	assert.Equal(t, want, got)

	body := strings.Repeat("11", 32) + "  other.tar.gz\n" + hexDigest + "  dist/x.tar.gz\n"
	got, err = parseChecksumFile(body, "x.tar.gz", "src", algoSHA256)
	require.NoError(t, err)
	assert.Equal(t, want, got)

	_, err = parseChecksumFile(body, "nope.tar.gz", "src", algoSHA256)
	require.ErrorIs(t, err, ErrNoArtifact)

	_, err = parseChecksumFile("nothex", "x", "src", algoSHA256)
	require.Error(t, err)

	// A sha512 file must hold a sha512 digest.
	_, err = parseChecksumFile(hexDigest, "x", "src", algoSHA512)
	require.ErrorContains(t, err, "expected a sha512 digest")
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
	def.Versions[0].StripPrefix = &empty
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	def.Versions = VersionSources{{Type: VersionsGitHubReleases, Repo: "acme/x"}}
	_, err = cli.ListVersions(t.Context(), def)
	require.NoError(t, err)

	assert.Equal(t, []string{"/index.json|", "/repos/acme/x/releases|Bearer secret"}, seen)
}

func TestJQNumbersKeepTheirDigits(t *testing.T) {
	code, err := compileJQ(".[]")
	require.NoError(t, err)
	var input any
	require.NoError(t, json.Unmarshal([]byte(`[20240101, 1000000, 1.10, "1.2", true, null, [7, "8"]]`), &input))
	got, err := runJQ(code, ".[]", input)
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
	assert.True(t, cli.IsInstalled(def, "1.2.0", testSystem))
	content, err := os.ReadFile(filepath.Join(cli.InstallDir("hello", "1.2.0", testSystem), "bin", "hello"))
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

	_, err := cli.Install(t.Context(), def, "1.2.0", testSystem, Artifact{Sha256: vendor.tarSHA})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exit status 3")
	assert.Contains(t, err.Error(), "something went wrong")
	assert.False(t, cli.IsInstalled(def, "1.2.0", testSystem), "no marker after a failed hook")

	info, err := os.Stat(cli.InstallDir("hello", "1.2.0", testSystem))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o755), info.Mode().Perm(), "install dir is world readable")
}

func TestRelativeFileURLsResolveAgainstDefinitionDir(t *testing.T) {
	projectDir := t.TempDir()
	defDir := filepath.Join(projectDir, "defs")
	require.NoError(t, os.MkdirAll(filepath.Join(defDir, "vendor"), 0o755))
	binary := []byte("#!/bin/sh\necho rel\n")
	require.NoError(t, os.WriteFile(filepath.Join(defDir, "vendor", "tool-1.0.0"), binary, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(defDir, "vendor", "SHA256SUMS"), []byte(sha256Hex(binary)+"  tool-1.0.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(defDir, "vendor", "versions.json"), []byte(`{"versions": ["1.0.0"]}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(defDir, "tool.pkg.json"), []byte(`{
	  "schema": 1, "name": "tool", "platforms": ["x86_64-linux"],
	  "versions": {"type": "http_json", "url": "file://vendor/versions.json", "select": ".versions[]"},
	  "install": {"type": "binary", "url": "file://vendor/tool-{{.Version}}",
	              "checksum": {"type": "sha256_url", "url": "file://vendor/SHA256SUMS"}}
	}`), 0o644))

	cli := NewClient()
	cli.CacheDir = t.TempDir()
	loaded, err := cli.Load(t.Context(), Ref{Source: "./defs/tool.pkg.json", Version: "latest"}, projectDir, "")
	require.NoError(t, err)

	resolved, err := cli.Resolve(t.Context(), loaded.Definition, "latest")
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", resolved.Version)
	// The lock keeps the relative URL so devbox.lock stays portable.
	want := "file://vendor/tool-1.0.0"
	assert.Equal(t, Artifact{URL: want, Sha256: sha256Hex(binary)}, resolved.Artifacts[testSystem])

	installed, err := cli.Install(t.Context(), loaded.Definition, "1.0.0", testSystem, resolved.Artifacts[testSystem])
	require.NoError(t, err)
	assert.Equal(t, want, installed.URL)
	content, err := os.ReadFile(filepath.Join(installed.Dir, "bin", "tool"))
	require.NoError(t, err)
	assert.Equal(t, binary, content)

	// Absolute file URLs and remote definitions are left alone.
	assert.Equal(t, "file:///abs/x", loaded.Definition.ResolveURL("file:///abs/x"))
	assert.Equal(t, "https://example.com/x", loaded.Definition.ResolveURL("https://example.com/x"))
	assert.Equal(t, "file://rel/x", (&Definition{}).ResolveURL("file://rel/x"))
}

func TestInstallRedoneWhenRecipeChanges(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh is required")
	}
	vendor := newFakeVendor(t)
	cli := vendor.client(t)
	def := vendor.archiveDef(t)
	art := Artifact{Sha256: vendor.tarSHA}

	first, err := cli.Install(t.Context(), def, "1.2.0", testSystem, art)
	require.NoError(t, err)
	assert.True(t, cli.IsInstalled(def, "1.2.0", testSystem))

	// Activation-only edits do not invalidate the install.
	def.Env = map[string]string{"X": "y"}
	assert.True(t, cli.IsInstalled(def, "1.2.0", testSystem))
	assert.Equal(t, def.InstallFingerprint(testSystem, "1.2.0"), def.InstallFingerprint(testSystem, "1.2.0"))

	// Changing the install recipe does: the hook now writes a different file.
	def.Hooks.PostInstall = []string{"echo redone > $DEVBOX_PKG_INSTALL_DIR/redone.txt"}
	assert.False(t, cli.IsInstalled(def, "1.2.0", testSystem))
	second, err := cli.Install(t.Context(), def, "1.2.0", testSystem, art)
	require.NoError(t, err)
	assert.Equal(t, first.Dir, second.Dir)
	_, err = os.Stat(filepath.Join(second.Dir, "redone.txt"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(second.Dir, "hook.txt"))
	assert.True(t, os.IsNotExist(err), "the old install was replaced, not layered on")
	assert.True(t, cli.IsInstalled(def, "1.2.0", testSystem))
	assert.Equal(t, 2, vendor.hitCount("/dl/hello-1.2.0-linux-amd64.tar.gz"))
}
