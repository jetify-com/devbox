// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// vendorServer is an httptest server with per-path hit counts.
type vendorServer struct {
	*httptest.Server
	mux  *http.ServeMux
	mu   sync.Mutex
	hits map[string]int
}

func newVendorServer(t *testing.T) *vendorServer {
	t.Helper()
	srv := &vendorServer{mux: http.NewServeMux(), hits: map[string]int{}}
	srv.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.mu.Lock()
		srv.hits[r.URL.RequestURI()]++
		srv.mu.Unlock()
		srv.mux.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func (srv *vendorServer) hitCount(uri string) int {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	return srv.hits[uri]
}

func (srv *vendorServer) client(t *testing.T) *Client {
	t.Helper()
	cli := NewClient()
	cli.HTTPClient = srv.Client()
	cli.CacheDir = filepath.Join(t.TempDir(), "pkgs")
	cli.GitHubAPIURL = srv.URL
	cli.GitHubToken = ""
	return cli
}

func parseDef(t *testing.T, format string, args ...any) *Definition {
	t.Helper()
	def, err := Parse([]byte(fmt.Sprintf(format, args...)))
	require.NoError(t, err)
	return def
}

func TestParseDigest(t *testing.T) {
	hex256 := strings.Repeat("ab", 32)
	hex512 := strings.Repeat("cd", 64)
	raw512, _ := hex.DecodeString(hex512)
	sri512 := "sha512-" + base64.StdEncoding.EncodeToString(raw512)

	tests := []struct {
		in, want string
		wantOK   bool
		wantAlgo string
	}{
		{hex256, "", true, algoSHA256},
		{strings.ToUpper(hex256), "", true, algoSHA256},
		{"sha256:" + hex256, "", true, algoSHA256},
		{hex512, "", true, algoSHA512},
		{"sha512:" + hex512, "", true, algoSHA512},
		{sri512, "", true, algoSHA512},
		{sri512 + " sha1-abc", "", true, algoSHA512}, // several SRI hashes
		{hex256, algoSHA512, false, ""},
		{"sha1-" + base64.StdEncoding.EncodeToString([]byte("0123456789abcdefghij")), "", false, ""},
		{strings.Repeat("ab", 20), "", false, ""}, // sha1 length
		{strings.Repeat("zz", 32), "", false, ""},
		{"sha512-not base64!", "", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := parseDigest(tt.in, "src", tt.want)
			if !tt.wantOK {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantAlgo, got.algo)
			if tt.wantAlgo == algoSHA512 {
				assert.Equal(t, hex512, got.hex)
			} else {
				assert.Equal(t, hex256, got.hex)
			}
		})
	}
}

// TestLockSkipsSystemsWithoutArtifacts locks a version that the vendor
// ships for one platform only: the other platform is left out of the
// lock, and a version with no artifact anywhere is an error.
func TestLockSkipsSystemsWithoutArtifacts(t *testing.T) {
	srv := newVendorServer(t)
	sha := strings.Repeat("ab", 32)
	srv.mux.HandleFunc("/dl/1.0.0/SHASUMS256.txt", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, "%s  x-1.0.0-linux-amd64.tar.gz\n", sha)
	})
	def := parseDef(t, `{
	  "schema": 1, "name": "x", "platforms": ["x86_64-linux", "aarch64-darwin"],
	  "versions": {"type": "static", "versions": ["1.0.0", "2.0.0"]},
	  "install": {
	    "type": "archive",
	    "url": "%[1]s/dl/{{.Version}}/x-{{.Version}}-{{.OS}}-{{.Arch}}.tar.gz",
	    "checksum": {"type": "sha256_url", "url": "%[1]s/dl/{{.Version}}/SHASUMS256.txt"}
	  }
	}`, srv.URL)
	cli := srv.client(t)

	artifacts, err := cli.Lock(t.Context(), def, "1.0.0")
	require.NoError(t, err)
	assert.Equal(t, map[string]Artifact{
		"x86_64-linux": {URL: srv.URL + "/dl/1.0.0/x-1.0.0-linux-amd64.tar.gz", Sha256: sha},
	}, artifacts)
	assert.Equal(t, 1, srv.hitCount("/dl/1.0.0/SHASUMS256.txt"), "the checksum file is fetched once for both systems")

	// 2.0.0 has no SHASUMS256.txt at all.
	_, err = cli.Lock(t.Context(), def, "2.0.0")
	require.ErrorIs(t, err, ErrNoArtifact)
	assert.Contains(t, err.Error(), "aarch64-darwin")
	assert.Contains(t, err.Error(), "x86_64-linux")
}

// TestLookup resolves artifacts through a zig-style index that maps
// version and platform to a URL and checksum.
func TestLookup(t *testing.T) {
	srv := newVendorServer(t)
	tarball := makeTarGz(t, map[string]string{"x-1.0.0/x": "#!/bin/sh\necho x 1.0.0\n"})
	sha := sha256Hex(tarball)
	srv.mux.HandleFunc("/index.json", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{
		  "1.0.0": {"x86_64-linux": {"tarball": "%[1]s/dl/x-1.0.0.tar.gz", "shasum": %[2]q}},
		  "0.9.0": {"x86_64-linux": {"tarball": "%[1]s/dl/x-0.9.0.tar.gz"}},
		  "0.8.0": {"x86_64-linux": {"tarball": null}}
		}`, srv.URL, sha)
	})
	srv.mux.HandleFunc("/dl/x-1.0.0.tar.gz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(tarball)
	})
	def := parseDef(t, `{
	  "schema": 1, "name": "x", "platforms": ["x86_64-linux", "aarch64-darwin"],
	  "os_map": {"linux": "linux"}, "arch_map": {"x86_64": "x86_64", "aarch64": "aarch64"},
	  "versions": {"type": "http_json", "url": "%[1]s/index.json", "select": "keys[]"},
	  "install": {
	    "type": "archive",
	    "lookup": {"url": "%[1]s/index.json", "select": ".[\"{{.Version}}\"][\"{{.System}}\"] // empty | {url: .tarball, checksum: .shasum}"},
	    "strip_components": 1
	  },
	  "bin": ["."],
	  "binaries": ["x"]
	}`, srv.URL)
	cli := srv.client(t)

	resolved, err := cli.Resolve(t.Context(), def, "1")
	require.NoError(t, err)
	assert.Equal(t, map[string]Artifact{
		"x86_64-linux": {URL: srv.URL + "/dl/x-1.0.0.tar.gz", Sha256: sha},
	}, resolved.Artifacts, "aarch64-darwin has no entry and is skipped")

	// An entry without a checksum is locked without one.
	artifacts, err := cli.Lock(t.Context(), def, "0.9.0")
	require.NoError(t, err)
	assert.Equal(t, Artifact{URL: srv.URL + "/dl/x-0.9.0.tar.gz"}, artifacts["x86_64-linux"])

	_, err = cli.Lock(t.Context(), def, "0.8.0")
	require.ErrorContains(t, err, `produced an object without "url"`)

	// Install without a locked artifact looks it up too, and verifies it.
	installed, err := cli.Install(t.Context(), def, "1.0.0", "x86_64-linux", Artifact{})
	require.NoError(t, err)
	assert.Equal(t, sha, installed.Sha256)
	assert.FileExists(t, filepath.Join(installed.Dir, "x"))

	assert.Equal(t, 1, srv.hitCount("/index.json"), "the index is fetched once per client")
}

func TestLookupIndexNotFound(t *testing.T) {
	srv := newVendorServer(t)
	def := parseDef(t, `{
	  "schema": 1, "name": "x", "platforms": ["x86_64-linux"],
	  "versions": {"type": "static", "versions": ["8.0.1"]},
	  "install": {
	    "type": "archive",
	    "lookup": {"url": "%s/feature/{{ index (splitList \".\" .Version) 0 }}.json", "select": ".[0]"}
	  }
	}`, srv.URL)
	_, err := srv.client(t).Lock(t.Context(), def, "8.0.1")
	require.ErrorIs(t, err, ErrNoArtifact)
	assert.Equal(t, 1, srv.hitCount("/feature/8.json"))
}

func TestChecksumGitHubRelease(t *testing.T) {
	srv := newVendorServer(t)
	sha := strings.Repeat("ef", 32)
	srv.mux.HandleFunc("/repos/acme/x/releases/tags/v1.0.0", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
		fmt.Fprintf(w, `{"assets": [
		  {"name": "x-linux-amd64.tar.gz", "digest": "sha256:%s"},
		  {"name": "x-darwin-arm64.tar.gz", "digest": null}
		]}`, sha)
	})
	def := parseDef(t, `{
	  "schema": 1, "name": "x", "platforms": ["x86_64-linux", "aarch64-darwin", "aarch64-linux"],
	  "versions": {"type": "static", "versions": ["1.0.0", "2.0.0"]},
	  "install": {
	    "type": "archive",
	    "url": "https://github.com/acme/x/releases/download/v{{.Version}}/x-{{.OS}}-{{.Arch}}.tar.gz",
	    "checksum": {"type": "github_release", "repo": "acme/x"}
	  }
	}`)
	cli := srv.client(t)
	cli.GitHubToken = "secret"

	artifacts, err := cli.Lock(t.Context(), def, "1.0.0")
	require.NoError(t, err)
	assert.Equal(t, sha, artifacts["x86_64-linux"].Sha256)
	assert.False(t, artifacts["aarch64-darwin"].HasDigest(), "an asset without a digest is locked without one")
	assert.NotContains(t, artifacts, "aarch64-linux", "a platform without an asset is skipped")
	assert.Equal(t, 1, srv.hitCount("/repos/acme/x/releases/tags/v1.0.0"), "the release is fetched once")

	_, err = cli.Lock(t.Context(), def, "2.0.0")
	require.ErrorIs(t, err, ErrNoArtifact, "a missing release means no artifacts")
}

func TestInstallVerifiesSHA512(t *testing.T) {
	srv := newVendorServer(t)
	tarball := makeTarGz(t, map[string]string{"package/bin/x": "#!/bin/sh\necho x\n"})
	sum := sha512.Sum512(tarball)
	integrity := "sha512-" + base64.StdEncoding.EncodeToString(sum[:])
	srv.mux.HandleFunc("/x/1.0.0", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"dist": {"integrity": %q}}`, integrity)
	})
	srv.mux.HandleFunc("/x/-/x-1.0.0.tgz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(tarball)
	})
	def := parseDef(t, `{
	  "schema": 1, "name": "x", "platforms": ["x86_64-linux"],
	  "versions": {"type": "static", "versions": ["1.0.0"]},
	  "install": {
	    "type": "archive",
	    "url": "%[1]s/x/-/x-{{.Version}}.tgz",
	    "checksum": {"type": "http_json", "url": "%[1]s/x/{{.Version}}", "select": ".dist.integrity"},
	    "strip_components": 1
	  },
	  "binaries": ["x"]
	}`, srv.URL)
	cli := srv.client(t)

	artifacts, err := cli.Lock(t.Context(), def, "1.0.0")
	require.NoError(t, err)
	artifact := artifacts["x86_64-linux"]
	assert.Equal(t, hex.EncodeToString(sum[:]), artifact.Sha512)
	assert.Empty(t, artifact.Sha256)

	bad := artifact
	bad.Sha512 = strings.Repeat("00", 64)
	_, err = cli.Install(t.Context(), def, "1.0.0", "x86_64-linux", bad)
	require.ErrorIs(t, err, ErrChecksumMismatch)
	assert.Contains(t, err.Error(), "sha512")

	installed, err := cli.Install(t.Context(), def, "1.0.0", "x86_64-linux", artifact)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(installed.Dir, "bin", "x"))

	// The marker remembers the sha512, so a lockfile that disagrees with
	// what is installed is caught without downloading again.
	_, err = cli.Install(t.Context(), def, "1.0.0", "x86_64-linux", bad)
	require.ErrorIs(t, err, ErrChecksumMismatch)
}

// TestBinaries checks that listed binaries are made executable (npm
// tarballs drop the bit) and that a definition whose binaries are missing
// fails to install.
func TestBinaries(t *testing.T) {
	srv := newVendorServer(t)
	tarball := makeTarGz(t, map[string]string{"bin/x": "#!/bin/sh\necho x\n"})
	srv.mux.HandleFunc("/x.tar.gz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(tarball)
	})
	const tmpl = `{
	  "schema": 1, "name": "x", "platforms": ["x86_64-linux"],
	  "versions": {"type": "static", "versions": ["1.0.0"]},
	  "install": {"type": "archive", "url": "%s/x.tar.gz"},
	  "binaries": %s
	}`

	cli := srv.client(t)
	def := parseDef(t, tmpl, srv.URL, `["x"]`)
	installed, err := cli.Install(t.Context(), def, "1.0.0", "x86_64-linux", Artifact{})
	require.NoError(t, err)
	info, err := os.Stat(filepath.Join(installed.Dir, "bin", "x"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())

	def = parseDef(t, tmpl, srv.URL, `["x", "y"]`)
	_, err = srv.client(t).Install(t.Context(), def, "1.0.0", "x86_64-linux", Artifact{})
	require.ErrorContains(t, err, "y not found")
}

// TestPostInstallWithoutPATH runs a hook while PATH is empty, as it is
// when devbox runs under testscript.
func TestPostInstallWithoutPATH(t *testing.T) {
	srv := newVendorServer(t)
	srv.mux.HandleFunc("/x", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("#!/bin/sh\n"))
	})
	def := parseDef(t, `{
	  "schema": 1, "name": "x", "platforms": ["x86_64-linux"],
	  "versions": {"type": "static", "versions": ["1.0.0"]},
	  "install": {"type": "binary", "url": "%s/x"},
	  "hooks": {"post_install": ["echo ok > hook.txt && ln -s hook.txt link.txt"]}
	}`, srv.URL)
	t.Setenv("PATH", "")
	installed, err := srv.client(t).Install(t.Context(), def, "1.0.0", "x86_64-linux", Artifact{})
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(installed.Dir, "link.txt"), "system utilities are on the hook's PATH")
}

// TestVersionSourceUnion resolves against two sources, like yarn 1 and
// yarn 2+ living in different npm packages.
func TestVersionSourceUnion(t *testing.T) {
	srv := newVendorServer(t)
	srv.mux.HandleFunc("/berry", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"versions": {"4.5.0": {}, "3.8.7": {}, "4.6.0-rc.1": {}}}`)
	})
	srv.mux.HandleFunc("/classic", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"versions": {"1.22.22": {}, "1.22.19": {}}}`)
	})
	def := parseDef(t, `{
	  "schema": 1, "name": "x", "platforms": ["x86_64-linux"],
	  "versions": [
	    {"type": "http_json", "url": "%[1]s/berry", "select": ".versions | keys[]"},
	    {"type": "http_json", "url": "%[1]s/classic", "select": ".versions | keys[]"}
	  ],
	  "install": {"type": "binary", "url": "https://example.com/x"}
	}`, srv.URL)
	cli := srv.client(t)

	for requested, want := range map[string]string{"latest": "4.5.0", "1": "1.22.22", "3": "3.8.7", "4.6.0-rc.1": "4.6.0-rc.1"} {
		got, err := cli.ResolveVersion(t.Context(), def, requested)
		require.NoError(t, err, requested)
		assert.Equal(t, want, got, requested)
	}
}

// TestHTTPJSONPagination pages through an Adoptium-style API that returns
// 404 past the last page.
func TestHTTPJSONPagination(t *testing.T) {
	srv := newVendorServer(t)
	pages := [][]string{{"27.0.1+0", "21.0.5+11"}, {"17.0.13+11", "11.0.25+9"}, {"8.0.432+6"}}
	srv.mux.HandleFunc("/versions", func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page >= len(pages) {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, `{"versions": [%s]}`, `"`+strings.Join(pages[page], `", "`)+`"`)
	})
	def := parseDef(t, `{
	  "schema": 1, "name": "x", "platforms": ["x86_64-linux"],
	  "versions": {"type": "http_json", "url": "%s/versions?page={{.Page}}", "select": ".versions[]"},
	  "install": {"type": "binary", "url": "https://example.com/x"}
	}`, srv.URL)

	// Resolution stops at the first page with a match.
	cli := srv.client(t)
	got, err := cli.ResolveVersion(t.Context(), def, "latest")
	require.NoError(t, err)
	assert.Equal(t, "27.0.1+0", got)
	assert.Equal(t, 0, srv.hitCount("/versions?page=1"))

	got, err = cli.ResolveVersion(t.Context(), def, "17")
	require.NoError(t, err)
	assert.Equal(t, "17.0.13+11", got)
	assert.Equal(t, 0, srv.hitCount("/versions?page=2"))

	// Listing reads every page until the 404.
	versions, err := cli.ListVersions(t.Context(), def)
	require.NoError(t, err)
	assert.Len(t, versions, 5)
	assert.Equal(t, 1, srv.hitCount("/versions?page=3"))
	assert.Equal(t, 0, srv.hitCount("/versions?page=4"))
}

// TestGitHubReleasesPagination reads more pages of releases only when the
// requested version is not on the first one.
func TestGitHubReleasesPagination(t *testing.T) {
	srv := newVendorServer(t)
	srv.mux.HandleFunc("/repos/acme/x/releases", func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		var tags []string
		switch page {
		case 0, 1: // a full page of 3.x
			for i := range githubPerPage {
				tags = append(tags, fmt.Sprintf(`{"tag_name": "v3.%d.0"}`, githubPerPage-i))
			}
		case 2:
			tags = []string{`{"tag_name": "v2.1.0"}`, `{"tag_name": "v1.0.0"}`}
		}
		fmt.Fprintf(w, "[%s]", strings.Join(tags, ","))
	})
	def := parseDef(t, `{
	  "schema": 1, "name": "x", "platforms": ["x86_64-linux"],
	  "versions": {"type": "github_releases", "repo": "acme/x"},
	  "install": {"type": "binary", "url": "https://example.com/x"}
	}`)
	cli := srv.client(t)

	got, err := cli.ResolveVersion(t.Context(), def, "latest")
	require.NoError(t, err)
	assert.Equal(t, "3.100.0", got)
	assert.Equal(t, 0, srv.hitCount("/repos/acme/x/releases?per_page=100&page=2"))

	got, err = cli.ResolveVersion(t.Context(), def, "2")
	require.NoError(t, err)
	assert.Equal(t, "2.1.0", got)
	assert.Equal(t, 1, srv.hitCount("/repos/acme/x/releases?per_page=100&page=2"))

	// A short page is the last one.
	_, err = cli.ResolveVersion(t.Context(), def, "9")
	require.ErrorIs(t, err, ErrVersionNotFound)
	assert.Equal(t, 0, srv.hitCount("/repos/acme/x/releases?per_page=100&page=3"))
}
