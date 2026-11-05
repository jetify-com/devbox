// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.jetify.com/devbox/nix/flake"
	"go.jetify.com/pkg/filecache"
)

const testCommit = "a1b2c3d4e5f6789012345678901234567890abcd"

// useTempPluginCaches points the remote plugin caches at a temp dir and
// removes any git clones made by the test.
func useTempPluginCaches(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	oldGit, oldGitHub := gitCache, githubCache
	gitCache = filecache.New("devbox/plugin/git", filecache.WithCacheDir[[]byte](dir))
	githubCache = filecache.New("devbox/plugin/github", filecache.WithCacheDir[[]byte](dir))
	RemoveGitClones()
	t.Cleanup(func() {
		gitCache, githubCache = oldGit, oldGitHub
		RemoveGitClones()
	})
}

func TestGetOrFetch(t *testing.T) {
	useTempPluginCaches(t)
	cache := gitCache
	errOffline := errors.New("offline")
	fetchOK := func(calls *int) func() ([]byte, error) {
		return func() ([]byte, error) {
			*calls++
			return []byte("new"), nil
		}
	}
	fetchFail := func(calls *int) func() ([]byte, error) {
		return func() ([]byte, error) {
			*calls++
			return nil, errOffline
		}
	}

	t.Run("fresh entry is used without fetching", func(t *testing.T) {
		require.NoError(t, cache.Set("fresh", []byte("old"), time.Hour))
		calls := 0
		got, err := getOrFetch(cache, "fresh", time.Hour, fetchFail(&calls))
		require.NoError(t, err)
		assert.Equal(t, "old", string(got))
		assert.Equal(t, 0, calls)
	})

	t.Run("expired entry is refreshed", func(t *testing.T) {
		require.NoError(t, cache.SetWithTime("refresh", []byte("old"), time.Now().Add(-time.Hour)))
		calls := 0
		got, err := getOrFetch(cache, "refresh", time.Hour, fetchOK(&calls))
		require.NoError(t, err)
		assert.Equal(t, "new", string(got))
		assert.Equal(t, 1, calls)

		// The refreshed value is cached.
		got, err = getOrFetch(cache, "refresh", time.Hour, fetchFail(&calls))
		require.NoError(t, err)
		assert.Equal(t, "new", string(got))
		assert.Equal(t, 1, calls)
	})

	t.Run("expired entry is used when fetch fails", func(t *testing.T) {
		require.NoError(t, cache.SetWithTime("stale", []byte("old"), time.Now().Add(-time.Hour)))
		calls := 0
		got, err := getOrFetch(cache, "stale", time.Hour, fetchFail(&calls))
		require.NoError(t, err)
		assert.Equal(t, "old", string(got))
		assert.Equal(t, 1, calls)
	})

	t.Run("fetch error is returned when nothing is cached", func(t *testing.T) {
		calls := 0
		_, err := getOrFetch(cache, "missing", time.Hour, fetchFail(&calls))
		require.ErrorIs(t, err, errOffline)
		assert.Equal(t, 1, calls)
	})
}

func TestRemoteCacheTTL(t *testing.T) {
	t.Setenv("DEVBOX_X_GITHUB_PLUGIN_CACHE_TTL", "")
	for rev, want := range map[string]time.Duration{
		"":         defaultRemoteCacheTTL,
		"abc1234":  defaultRemoteCacheTTL, // short hash
		"main":     defaultRemoteCacheTTL,
		testCommit: pinnedCacheTTL,
		"A1B2C3D4E5F6789012345678901234567890ABCD": pinnedCacheTTL,
	} {
		got, err := remoteCacheTTL(rev)
		require.NoError(t, err)
		assert.Equal(t, want, got, "rev %q", rev)
	}

	// The env var only overrides the TTL for content that isn't pinned to a
	// commit.
	t.Setenv("DEVBOX_X_GITHUB_PLUGIN_CACHE_TTL", "1h")
	got, err := remoteCacheTTL("main")
	require.NoError(t, err)
	assert.Equal(t, time.Hour, got)
	got, err = remoteCacheTTL(testCommit)
	require.NoError(t, err)
	assert.Equal(t, pinnedCacheTTL, got)
}

// fakeGitRemote stubs out git clones. Each clone writes files into a new
// temp dir.
type fakeGitRemote struct {
	files  map[string]string
	clones int
	err    error
}

func stubGitClone(t *testing.T, files map[string]string) *fakeGitRemote {
	t.Helper()
	remote := &fakeGitRemote{files: files}
	old := cloneGitRepo
	cloneGitRepo = func(*gitPlugin) (string, error) {
		remote.clones++
		if remote.err != nil {
			return "", remote.err
		}
		dir := t.TempDir()
		for name, content := range remote.files {
			p := filepath.Join(dir, name)
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				return "", err
			}
			if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
				return "", err
			}
		}
		return dir, nil
	}
	t.Cleanup(func() { cloneGitRepo = old })
	return remote
}

func newTestGitPlugin(ref, rev, dir string) *gitPlugin {
	return &gitPlugin{
		ref: &flake.Ref{
			Type: flake.TypeGit,
			URL:  "https://example.com/org/plugins.git",
			Ref:  ref,
			Rev:  rev,
			Dir:  dir,
		},
		name: "test",
	}
}

func readGitFile(t *testing.T, p *gitPlugin, subpath string) string {
	t.Helper()
	content, err := p.FileContent(subpath)
	require.NoError(t, err)
	return string(content)
}

func TestGitPluginClonesOncePerRepoRef(t *testing.T) {
	useTempPluginCaches(t)
	remote := stubGitClone(t, map[string]string{
		"plugin.json":     "root plugin",
		"a.txt":           "a",
		"sub/plugin.json": "sub plugin",
		"sub/b.txt":       "b",
	})

	root := newTestGitPlugin("main", "", "")
	assert.Equal(t, "root plugin", readGitFile(t, root, "plugin.json"))
	assert.Equal(t, "a", readGitFile(t, root, "a.txt"))

	// Another plugin in the same repo and ref reuses the clone.
	sub := newTestGitPlugin("main", "", "sub")
	assert.Equal(t, "sub plugin", readGitFile(t, sub, "plugin.json"))
	assert.Equal(t, "b", readGitFile(t, sub, "b.txt"))
	assert.Equal(t, 1, remote.clones)

	// A different ref needs its own clone.
	other := newTestGitPlugin("dev", "", "")
	assert.Equal(t, "a", readGitFile(t, other, "a.txt"))
	assert.Equal(t, 2, remote.clones)
}

func TestGitPluginCommitPinnedDoesNotExpire(t *testing.T) {
	useTempPluginCaches(t)
	t.Setenv("DEVBOX_X_GITHUB_PLUGIN_CACHE_TTL", "1ns")
	remote := stubGitClone(t, map[string]string{"plugin.json": "v1"})

	p := newTestGitPlugin("main", testCommit, "")
	assert.Equal(t, "v1", readGitFile(t, p, "plugin.json"))

	// Even with a new process (no clones) and a 1ns TTL override, the
	// pinned content comes from the cache without cloning again.
	RemoveGitClones()
	remote.files["plugin.json"] = "v2"
	assert.Equal(t, "v1", readGitFile(t, p, "plugin.json"))
	assert.Equal(t, 1, remote.clones)
}

func TestGitPluginUnpinnedExpires(t *testing.T) {
	for name, rev := range map[string]string{
		"branch":     "",
		"short hash": "a1b2c3d",
	} {
		t.Run(name, func(t *testing.T) {
			useTempPluginCaches(t)
			t.Setenv("DEVBOX_X_GITHUB_PLUGIN_CACHE_TTL", "1ns")
			remote := stubGitClone(t, map[string]string{"plugin.json": "v1"})

			p := newTestGitPlugin("main", rev, "")
			assert.Equal(t, "v1", readGitFile(t, p, "plugin.json"))

			RemoveGitClones()
			remote.files["plugin.json"] = "v2"
			assert.Equal(t, "v2", readGitFile(t, p, "plugin.json"))
			assert.Equal(t, 2, remote.clones)
		})
	}
}

func TestGitPluginExpiredFallsBackWhenCloneFails(t *testing.T) {
	useTempPluginCaches(t)
	t.Setenv("DEVBOX_X_GITHUB_PLUGIN_CACHE_TTL", "1ns")
	remote := stubGitClone(t, map[string]string{"plugin.json": "v1"})

	p := newTestGitPlugin("main", "", "")
	assert.Equal(t, "v1", readGitFile(t, p, "plugin.json"))

	RemoveGitClones()
	remote.err = errors.New("network is unreachable")
	assert.Equal(t, "v1", readGitFile(t, p, "plugin.json"))
	assert.Equal(t, 2, remote.clones)
}

func TestGitPluginCloneFailsWithNothingCached(t *testing.T) {
	useTempPluginCaches(t)
	remote := stubGitClone(t, nil)
	remote.err = errors.New("network is unreachable")

	p := newTestGitPlugin("main", "", "")
	_, err := p.FileContent("plugin.json")
	require.ErrorIs(t, err, remote.err)

	// The failed clone isn't retried for other files in the same process.
	_, err = p.FileContent("a.txt")
	require.ErrorIs(t, err, remote.err)
	assert.Equal(t, 1, remote.clones)
}

// fakeGitHub serves plugin files over HTTP. Each response body is the
// requested file name and the current version.
type fakeGitHub struct {
	server   *httptest.Server
	requests atomic.Int32
	version  atomic.Int32
	status   atomic.Int32
}

func stubGitHub(t *testing.T) *fakeGitHub {
	t.Helper()
	fake := &fakeGitHub{}
	fake.version.Store(1)
	fake.status.Store(http.StatusOK)
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fake.requests.Add(1)
		w.WriteHeader(int(fake.status.Load()))
		fmt.Fprintf(w, "%s v%d", path.Base(r.URL.Path), fake.version.Load())
	}))
	t.Cleanup(fake.server.Close)

	old := githubContentBaseURL
	githubContentBaseURL = fake.server.URL + "/"
	t.Cleanup(func() { githubContentBaseURL = old })
	return fake
}

func newTestGithubPlugin(ref, rev string) *githubPlugin {
	return &githubPlugin{
		ref: flake.Ref{
			Type:  flake.TypeGitHub,
			Owner: "org",
			Repo:  "plugins",
			Ref:   ref,
			Rev:   rev,
			Dir:   "mongodb",
		},
		name: "test",
	}
}

func readGithubPluginJSON(t *testing.T, plugin *githubPlugin) string {
	t.Helper()
	content, err := plugin.FileContent("plugin.json")
	require.NoError(t, err)
	return string(content)
}

func TestGithubPluginHTTPClientHasTimeout(t *testing.T) {
	assert.Positive(t, githubHTTPClient.Timeout)
}

func TestGithubPluginCommitPinnedDoesNotExpire(t *testing.T) {
	useTempPluginCaches(t)
	t.Setenv("DEVBOX_X_GITHUB_PLUGIN_CACHE_TTL", "1ns")
	fake := stubGitHub(t)

	plugin := newTestGithubPlugin("", testCommit)
	assert.Equal(t, "plugin.json v1", readGithubPluginJSON(t, plugin))
	fake.version.Store(2)
	assert.Equal(t, "plugin.json v1", readGithubPluginJSON(t, plugin))
	assert.Equal(t, int32(1), fake.requests.Load())
}

func TestGithubPluginBranchExpires(t *testing.T) {
	useTempPluginCaches(t)
	t.Setenv("DEVBOX_X_GITHUB_PLUGIN_CACHE_TTL", "1ns")
	fake := stubGitHub(t)

	plugin := newTestGithubPlugin("main", "")
	assert.Equal(t, "plugin.json v1", readGithubPluginJSON(t, plugin))
	fake.version.Store(2)
	assert.Equal(t, "plugin.json v2", readGithubPluginJSON(t, plugin))
	assert.Equal(t, int32(2), fake.requests.Load())
}

func TestGithubPluginExpiredFallsBackWhenFetchFails(t *testing.T) {
	useTempPluginCaches(t)
	t.Setenv("DEVBOX_X_GITHUB_PLUGIN_CACHE_TTL", "1ns")
	fake := stubGitHub(t)

	plugin := newTestGithubPlugin("main", "")
	assert.Equal(t, "plugin.json v1", readGithubPluginJSON(t, plugin))

	// Server error.
	fake.status.Store(http.StatusInternalServerError)
	assert.Equal(t, "plugin.json v1", readGithubPluginJSON(t, plugin))

	// Network error.
	fake.server.Close()
	assert.Equal(t, "plugin.json v1", readGithubPluginJSON(t, plugin))

	// Nothing cached for this file, so the error is returned.
	_, err := plugin.FileContent("other.txt")
	require.Error(t, err)
}
