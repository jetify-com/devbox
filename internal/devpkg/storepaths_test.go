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

func (fake *fakeNix) query(installables, resolved map[string][]string, secure, insecure []string) *storePathQuery[string] {
	return &storePathQuery[string]{
		resolved:     resolved,
		secure:       secure,
		insecure:     insecure,
		installables: installables,
		pathInfo: func(_ context.Context, inst []string, _ bool) ([]string, error) {
			fake.pathInfoArgs = append(fake.pathInfoArgs, inst)
			paths := []string{}
			for _, i := range inst {
				if fake.failing[i] {
					return nil, errors.New("eval failed: " + i)
				}
				paths = append(paths, "/nix/store/"+strings.ReplaceAll(i, "#", "-"))
			}
			return paths, nil
		},
		inStore: func(_ context.Context, paths []string) (map[string]bool, error) {
			m := map[string]bool{}
			for _, p := range paths {
				m[p] = fake.inStore[p]
			}
			return m, nil
		},
		single: func(_ context.Context, item string) ([]string, error) {
			fake.singleCalls = append(fake.singleCalls, item)
			paths := []string{}
			for _, i := range installables[item] {
				if fake.failing[i] {
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
	fake := &fakeNix{inStore: map[string]bool{
		"/nix/store/a-out": true, "/nix/store/a-man": true,
		"/nix/store/b-out": true, "/nix/store/c-out": true,
		"/nix/store/resolved": true,
	}}
	query := fake.query(testInstallables, map[string][]string{"r": {"/nix/store/resolved"}}, []string{"a", "b", "c"}, nil)
	missing, err := query.notInStore(context.Background(), []string{"r", "a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 0 {
		t.Errorf("got missing %v, want none", missing)
	}
	if len(fake.pathInfoArgs) != 1 || len(fake.pathInfoArgs[0]) != 4 {
		t.Errorf("got path-info calls %v, want one call with all 4 installables", fake.pathInfoArgs)
	}
	if len(fake.singleCalls) != 0 {
		t.Errorf("got single calls %v, want none", fake.singleCalls)
	}
}

func TestNotInStoreMissingPathIsAttributed(t *testing.T) {
	fake := &fakeNix{inStore: map[string]bool{
		"/nix/store/a-out": true, // a-man is missing
		"/nix/store/b-out": true,
		// resolved r is missing
	}}
	query := fake.query(testInstallables, map[string][]string{"r": {"/nix/store/resolved"}}, []string{"a", "b"}, []string{"c"})
	missing, err := query.notInStore(context.Background(), []string{"r", "a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"r", "a", "c"}; !slices.Equal(missing, want) {
		t.Errorf("got missing %v, want %v", missing, want)
	}
	// The single insecure item skips the batch call.
	if len(fake.pathInfoArgs) != 1 {
		t.Errorf("got path-info calls %v, want 1", fake.pathInfoArgs)
	}
	if want := []string{"c", "a", "b"}; !slices.Equal(fake.singleCalls, want) {
		t.Errorf("got single calls %v, want %v", fake.singleCalls, want)
	}
}

func TestNotInStoreBatchErrorIsAttributed(t *testing.T) {
	fake := &fakeNix{failing: map[string]bool{"b#out": true}}
	query := fake.query(testInstallables, nil, []string{"a", "b", "c"}, nil)
	_, err := query.notInStore(context.Background(), []string{"a", "b", "c"})
	if err == nil || err.Error() != "error installing package b" {
		t.Errorf("got error %v, want error attributed to package b", err)
	}
	if want := []string{"a", "b"}; !slices.Equal(fake.singleCalls, want) {
		t.Errorf("got single calls %v, want %v", fake.singleCalls, want)
	}
}
