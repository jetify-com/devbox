// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lock

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.jetify.com/devbox/internal/envir"
	"go.jetify.com/devbox/internal/nix"
	"go.jetify.com/devbox/internal/searcher"
)

// fakeResolveServer serves /v1/resolve. It counts requests per package name,
// and its handler blocks until unblock is called, so tests can control when
// requests finish.
type fakeResolveServer struct {
	*httptest.Server

	mu       sync.Mutex
	requests map[string]int
	inFlight chan string
	release  chan struct{}
	// unblock releases blocked handlers. It's safe to call more than once.
	unblock func()
}

func newFakeResolveServer(t *testing.T) *fakeResolveServer {
	t.Helper()
	srv := &fakeResolveServer{
		requests: map[string]int{},
		inFlight: make(chan string, 100),
		release:  make(chan struct{}),
	}
	srv.unblock = sync.OnceFunc(func() { close(srv.release) })
	srv.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		srv.mu.Lock()
		srv.requests[name]++
		srv.mu.Unlock()
		srv.inFlight <- name
		<-srv.release

		info := searcher.PackageInfo{
			CommitHash: "1111111111111111111111111111111111111111",
			AttrPaths:  []string{name},
			Version:    r.URL.Query().Get("version"),
		}
		_ = json.NewEncoder(w).Encode(searcher.PackageVersion{
			PackageInfo: info,
			Name:        name,
			// No store hash, so resolving doesn't shell out to nix.
			Systems: map[string]searcher.PackageInfo{"x86_64-linux": info},
		})
	}))
	t.Cleanup(srv.Close)
	// Cleanups run in reverse order, so this releases any handlers still
	// blocked (e.g. after a t.Fatal) before Close waits for them.
	t.Cleanup(srv.unblock)
	t.Setenv(envir.DevboxSearchHost, srv.URL)
	t.Setenv("DEVBOX_FEATURE_RESOLVE_V2", "0")
	return srv
}

func (s *fakeResolveServer) requestCount(name string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requests[name]
}

// TestResolveConcurrent checks that concurrent Resolve calls for different
// packages make their search requests in parallel, that concurrent calls for
// the same package share one request, and that every caller gets the stored
// entry.
func TestResolveConcurrent(t *testing.T) {
	srv := newFakeResolveServer(t)
	lockfile := &File{Packages: map[string]*Package{}}

	names := []string{"hello", "jq", "ripgrep", "curl", "git"}
	const callersPerPkg = 4

	type result struct {
		pkg   string
		entry *Package
		err   error
	}
	results := make(chan result, len(names)*callersPerPkg)
	var callers sync.WaitGroup
	for _, name := range names {
		pkg := name + "@1.0"
		for range callersPerPkg {
			callers.Go(func() {
				entry, err := lockfile.Resolve(pkg)
				results <- result{pkg, entry, err}
			})
		}
	}

	// Every package's request must be in flight at the same time. If Resolve
	// held the lockfile lock during the request, only one would arrive and
	// this would time out.
	seen := map[string]bool{}
	timeout := time.After(10 * time.Second)
	for len(seen) < len(names) {
		select {
		case name := <-srv.inFlight:
			seen[name] = true
		case <-timeout:
			t.Fatalf("got %d concurrent requests, want %d: requests are serialized", len(seen), len(names))
		}
	}

	// While the requests are blocked, the lockfile must stay usable.
	if lockfile.Get("hello@1.0") != nil {
		t.Error("Get returned an entry for a package that is still resolving")
	}
	_ = lockfile.HasAllowInsecurePackages()

	srv.unblock()
	callers.Wait()
	close(results)

	for r := range results {
		if r.err != nil {
			t.Fatalf("Resolve(%q): %v", r.pkg, r.err)
		}
		if got := lockfile.Get(r.pkg); got != r.entry {
			t.Errorf("Resolve(%q) returned %p, but the lockfile has %p", r.pkg, r.entry, got)
		}
		name := r.pkg[:len(r.pkg)-len("@1.0")]
		want := fmt.Sprintf("github:NixOS/nixpkgs/1111111111111111111111111111111111111111#%s", name)
		if r.entry.Resolved != want {
			t.Errorf("Resolve(%q).Resolved = %q, want %q", r.pkg, r.entry.Resolved, want)
		}
	}
	for _, name := range names {
		if got := srv.requestCount(name); got != 1 {
			t.Errorf("got %d search requests for %q, want 1", got, name)
		}
	}

	// Resolving again uses the lockfile and makes no requests.
	if _, err := lockfile.Resolve("hello@1.0"); err != nil {
		t.Fatal(err)
	}
	if got := srv.requestCount("hello"); got != 1 {
		t.Errorf("got %d search requests for hello after re-resolving, want 1", got)
	}
}

// TestResolveConcurrentError checks that a failed resolve is returned to every
// concurrent caller, isn't stored, and is retried by the next call.
func TestResolveConcurrentError(t *testing.T) {
	var fail atomic.Bool
	fail.Store(true)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		info := searcher.PackageInfo{
			CommitHash: "1111111111111111111111111111111111111111",
			AttrPaths:  []string{"hello"},
			Version:    "1.0",
		}
		_ = json.NewEncoder(w).Encode(searcher.PackageVersion{
			PackageInfo: info,
			Name:        "hello",
			Systems:     map[string]searcher.PackageInfo{"x86_64-linux": info},
		})
	}))
	t.Cleanup(srv.Close)
	t.Setenv(envir.DevboxSearchHost, srv.URL)
	t.Setenv("DEVBOX_FEATURE_RESOLVE_V2", "0")

	lockfile := &File{Packages: map[string]*Package{}}
	var callers sync.WaitGroup
	errs := make(chan error, 8)
	for range 8 {
		callers.Go(func() {
			_, err := lockfile.Resolve("hello@1.0")
			errs <- err
		})
	}
	callers.Wait()
	close(errs)
	for err := range errs {
		if !errors.Is(err, nix.ErrPackageNotFound) {
			t.Errorf("got error %v, want nix.ErrPackageNotFound", err)
		}
	}
	if lockfile.Get("hello@1.0") != nil {
		t.Error("failed resolve was stored in the lockfile")
	}

	fail.Store(false)
	entry, err := lockfile.Resolve("hello@1.0")
	if err != nil {
		t.Fatalf("Resolve after failure: %v", err)
	}
	if lockfile.Get("hello@1.0") != entry {
		t.Error("successful resolve wasn't stored in the lockfile")
	}
}
