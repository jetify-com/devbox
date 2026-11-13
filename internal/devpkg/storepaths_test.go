// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devpkg

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

// fakeNix simulates `nix path-info` for installables of the form
// "<item>#<output>", which produce the store path "/nix/store/<item>-<output>".
type fakeNix struct {
	inStore      map[string]bool
	failing      map[string]bool // installables that fail to evaluate
	pathInfoArgs [][]string
	singleCalls  []string
}

func (f *fakeNix) query(installables map[string][]string, resolved map[string][]string, secure, insecure []string) *storePathQuery[string] {
	return &storePathQuery[string]{
		resolved:     resolved,
		secure:       secure,
		insecure:     insecure,
		installables: installables,
		pathInfo: func(_ context.Context, inst []string, _ bool) ([]string, error) {
			f.pathInfoArgs = append(f.pathInfoArgs, inst)
			paths := []string{}
			for _, i := range inst {
				if f.failing[i] {
					return nil, errors.New("eval failed: " + i)
				}
				paths = append(paths, "/nix/store/"+strings.ReplaceAll(i, "#", "-"))
			}
			return paths, nil
		},
		inStore: func(_ context.Context, paths []string) (map[string]bool, error) {
			m := map[string]bool{}
			for _, p := range paths {
				m[p] = f.inStore[p]
			}
			return m, nil
		},
		single: func(_ context.Context, item string) ([]string, error) {
			f.singleCalls = append(f.singleCalls, item)
			paths := []string{}
			for _, i := range installables[item] {
				if f.failing[i] {
					return nil, errors.New("error installing package " + item)
				}
				paths = append(paths, "/nix/store/"+strings.ReplaceAll(i, "#", "-"))
			}
			return paths, nil
		},
	}
}

var testInstallables = map[string][]string{
	"a": {"a#out", "a#man"},
	"b": {"b#out"},
	"c": {"c#out"},
}

func TestNotInStoreAllInstalledUsesOneBatch(t *testing.T) {
	f := &fakeNix{inStore: map[string]bool{
		"/nix/store/a-out": true, "/nix/store/a-man": true,
		"/nix/store/b-out": true, "/nix/store/c-out": true,
		"/nix/store/resolved": true,
	}}
	q := f.query(testInstallables, map[string][]string{"r": {"/nix/store/resolved"}}, []string{"a", "b", "c"}, nil)
	missing, err := q.notInStore(context.Background(), []string{"r", "a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 0 {
		t.Errorf("got missing %v, want none", missing)
	}
	if len(f.pathInfoArgs) != 1 || len(f.pathInfoArgs[0]) != 4 {
		t.Errorf("got path-info calls %v, want one call with all 4 installables", f.pathInfoArgs)
	}
	if len(f.singleCalls) != 0 {
		t.Errorf("got single calls %v, want none", f.singleCalls)
	}
}

func TestNotInStoreMissingPathIsAttributed(t *testing.T) {
	f := &fakeNix{inStore: map[string]bool{
		"/nix/store/a-out": true, // a-man is missing
		"/nix/store/b-out": true,
		// resolved r is missing
	}}
	q := f.query(testInstallables, map[string][]string{"r": {"/nix/store/resolved"}}, []string{"a", "b"}, []string{"c"})
	missing, err := q.notInStore(context.Background(), []string{"r", "a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"r", "a", "c"}; !slices.Equal(missing, want) {
		t.Errorf("got missing %v, want %v", missing, want)
	}
	// The single insecure item skips the batch call.
	if len(f.pathInfoArgs) != 1 {
		t.Errorf("got path-info calls %v, want 1", f.pathInfoArgs)
	}
	if want := []string{"c", "a", "b"}; !slices.Equal(f.singleCalls, want) {
		t.Errorf("got single calls %v, want %v", f.singleCalls, want)
	}
}

func TestNotInStoreBatchErrorIsAttributed(t *testing.T) {
	f := &fakeNix{failing: map[string]bool{"b#out": true}}
	q := f.query(testInstallables, nil, []string{"a", "b", "c"}, nil)
	_, err := q.notInStore(context.Background(), []string{"a", "b", "c"})
	if err == nil || err.Error() != "error installing package b" {
		t.Errorf("got error %v, want error attributed to package b", err)
	}
	if want := []string{"a", "b"}; !slices.Equal(f.singleCalls, want) {
		t.Errorf("got single calls %v, want %v", f.singleCalls, want)
	}
}
