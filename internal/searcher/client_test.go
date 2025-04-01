// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package searcher

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestExecGetRetries(t *testing.T) {
	origBackoff := retryBackoff
	retryBackoff = func(int) time.Duration { return 0 }
	t.Cleanup(func() { retryBackoff = origBackoff })

	type result struct{ Name string }
	tests := []struct {
		name         string
		statuses     []int // status per attempt; the last one repeats
		wantRequests int32
		wantErr      error // nil means success
		wantAnyErr   bool
	}{
		{name: "success", statuses: []int{200}, wantRequests: 1},
		{name: "recovers after 5xx", statuses: []int{500, 503, 200}, wantRequests: 3},
		{name: "recovers after 429", statuses: []int{429, 200}, wantRequests: 2},
		{name: "gives up after retries", statuses: []int{502}, wantRequests: maxRetries + 1, wantErr: ErrUnavailable},
		{name: "404 is not retried", statuses: []int{404}, wantRequests: 1, wantErr: ErrNotFound},
		{name: "400 is not retried", statuses: []int{400}, wantRequests: 1, wantAnyErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var requests atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				n := int(requests.Add(1))
				status := test.statuses[min(n, len(test.statuses))-1]
				w.WriteHeader(status)
				if status == 200 {
					_, _ = w.Write([]byte(`{"Name":"hello"}`))
				}
			}))
			defer srv.Close()

			got, err := execGet[result](context.Background(), srv.URL)
			if n := requests.Load(); n != test.wantRequests {
				t.Errorf("got %d requests, want %d", n, test.wantRequests)
			}
			switch {
			case test.wantErr != nil:
				if !errors.Is(err, test.wantErr) {
					t.Errorf("got error %v, want %v", err, test.wantErr)
				}
			case test.wantAnyErr:
				if err == nil || errors.Is(err, ErrUnavailable) {
					t.Errorf("got error %v, want a non-retryable error", err)
				}
			default:
				if err != nil {
					t.Fatalf("got error %v", err)
				}
				if got.Name != "hello" {
					t.Errorf("got %+v", got)
				}
			}
		})
	}
}

func TestExecGetNetworkErrorIsUnavailable(t *testing.T) {
	origBackoff := retryBackoff
	retryBackoff = func(int) time.Duration { return 0 }
	t.Cleanup(func() { retryBackoff = origBackoff })

	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // Nothing is listening anymore.

	_, err := execGet[struct{}](context.Background(), url)
	if !errors.Is(err, ErrUnavailable) {
		t.Errorf("got error %v, want ErrUnavailable", err)
	}
}

func TestExecGetStopsRetryingWhenCanceled(t *testing.T) {
	var requests atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		cancel()
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := execGet[struct{}](ctx, srv.URL)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("got error %v, want context.Canceled", err)
	}
	if n := requests.Load(); n != 1 {
		t.Errorf("got %d requests, want 1", n)
	}
}

func TestSearchInlineReleases(t *testing.T) {
	var gotReleases string
	var pkgCalls atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/search", func(w http.ResponseWriter, r *http.Request) {
		gotReleases = r.URL.Query().Get("releases")
		_, _ = w.Write([]byte(`{"total_results": 3, "results": [
			{"name": "ripgrep", "releases": [
				{"version": "15.0.0", "systems": ["x86_64-linux", "aarch64-darwin"]},
				{"version": "14.1.1", "systems": []}
			]},
			{"name": "ripgrep-all", "releases": []},
			{"name": "past-limit", "releases": []}
		]}`))
	})
	mux.HandleFunc("/v2/pkg", func(w http.ResponseWriter, r *http.Request) {
		pkgCalls.Add(1)
		http.NotFound(w, r)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	got, err := (&client{host: server.URL}).Search(t.Context(), "ripgrep", 2)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if gotReleases != "2" {
		t.Errorf("releases = %q, want %q", gotReleases, "2")
	}
	if n := pkgCalls.Load(); n != 0 {
		t.Errorf("made %d /v2/pkg calls, want 0", n)
	}
	want := &SearchResults{
		NumResults: 3,
		Packages: []Package{
			{Name: "ripgrep", Versions: []SearchRelease{
				{Version: "15.0.0", Systems: []string{"aarch64-darwin", "x86_64-linux"}},
				{Version: "14.1.1", Systems: []string{}},
			}},
			{Name: "ripgrep-all", Versions: []SearchRelease{}},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Search() mismatch (-want +got):\n%s", diff)
	}
}

func TestSearchAllReleases(t *testing.T) {
	var gotReleases string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReleases = r.URL.Query().Get("releases")
		_, _ = w.Write([]byte(`{"total_results": 0, "results": []}`))
	}))
	t.Cleanup(server.Close)

	if _, err := (&client{host: server.URL}).Search(t.Context(), "go", 0); err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if gotReleases != "all" {
		t.Errorf("releases = %q, want %q", gotReleases, "all")
	}
}

func TestSearchFallsBackToPkg(t *testing.T) {
	origBackoff := retryBackoff
	retryBackoff = func(int) time.Duration { return 0 }
	t.Cleanup(func() { retryBackoff = origBackoff })

	var lookedMu sync.Mutex
	var looked []string
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/search", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"total_results": 14, "results": [
			{"name": "ripgrep"}, {"name": "removed"}, {"name": "broken"},
			{"name": "ripgrep-all"}, {"name": "past-limit"}
		]}`))
	})
	mux.HandleFunc("/v2/pkg", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		lookedMu.Lock()
		looked = append(looked, name)
		lookedMu.Unlock()
		switch name {
		case "ripgrep":
			_, _ = w.Write([]byte(`{"name": "ripgrep", "releases": [
				{"version": "15.0.0", "platforms": [
					{"arch": "x86-64", "os": "Linux", "attribute_path": "ripgrep"},
					{"arch": "arm64", "os": "macOS", "attribute_path": "ripgrep"},
					{"arch": "x86-64", "os": "Linux", "attribute_path": "rg"}
				]},
				{"version": "14.1.1", "platforms": []},
				{"version": "13.0.0", "platforms": []},
				{"version": "12.1.1", "platforms": []},
				{"version": "11.0.2", "platforms": []}
			]}`))
		case "ripgrep-all":
			_, _ = w.Write([]byte(`{"name": "ripgrep-all", "releases": [
				{"version": "0.10.10", "platforms": [{"arch": "x86-64", "os": "Linux"}]}
			]}`))
		case "broken":
			http.Error(w, "boom", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	got, err := (&client{host: server.URL}).Search(t.Context(), "ripgrep", 4)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	slices.Sort(looked)
	looked = slices.Compact(looked)
	if diff := cmp.Diff([]string{"broken", "removed", "ripgrep", "ripgrep-all"}, looked); diff != "" {
		t.Errorf("/v2/pkg lookups mismatch (-want +got):\n%s", diff)
	}
	want := &SearchResults{
		NumResults: 14,
		Packages: []Package{
			{Name: "ripgrep", Versions: []SearchRelease{
				{Version: "15.0.0", Systems: []string{"aarch64-darwin", "x86_64-linux"}},
				{Version: "14.1.1", Systems: []string{}},
				{Version: "13.0.0", Systems: []string{}},
				{Version: "12.1.1", Systems: []string{}},
			}},
			{Name: "ripgrep-all", Versions: []SearchRelease{
				{Version: "0.10.10", Systems: []string{"x86_64-linux"}},
			}},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Search() mismatch (-want +got):\n%s", diff)
	}
}

func TestSearchFailsWhenEveryLookupFails(t *testing.T) {
	origBackoff := retryBackoff
	retryBackoff = func(int) time.Duration { return 0 }
	t.Cleanup(func() { retryBackoff = origBackoff })

	mux := http.NewServeMux()
	mux.HandleFunc("/v2/search", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"total_results": 2, "results": [{"name": "removed"}, {"name": "broken"}]}`))
	})
	mux.HandleFunc("/v2/pkg", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("name") == "broken" {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		http.NotFound(w, r)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	if _, err := (&client{host: server.URL}).Search(t.Context(), "q", 0); err == nil {
		t.Error("Search() error = nil, want the failed lookup's error")
	}
}

func TestSearchNoResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"total_results": 0, "results": []}`))
	}))
	t.Cleanup(server.Close)

	got, err := (&client{host: server.URL}).Search(t.Context(), "nothing", 0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(got.Packages) != 0 {
		t.Errorf("Search() packages = %v, want none", got.Packages)
	}
}
