// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devpkg

import (
	"context"
	"io"

	"github.com/samber/lo"

	"go.jetify.com/devbox/internal/boxcli/featureflag"
	"go.jetify.com/devbox/internal/debug"
	"go.jetify.com/devbox/internal/nix"
	"go.jetify.com/devbox/internal/ux"
)

// PackagesNotInStore returns the packages that have at least one output that
// isn't in the local nix store, in the order they appear in pkgs.
//
// Packages with store paths in the lockfile are checked directly. Others
// need `nix path-info` to evaluate their installables; those are batched into
// a single call per allow_insecure setting instead of one call per
// installable.
func PackagesNotInStore(ctx context.Context, w io.Writer, pkgs []*Package) ([]*Package, error) {
	defer debug.FunctionTimer().End()

	q := storePathQuery[*Package]{
		resolved:     map[*Package][]string{},
		installables: map[*Package][]string{},
		pathInfo:     nix.StorePathsFromInstallables,
		inStore:      nix.StorePathsAreInStore,
		single: func(ctx context.Context, p *Package) ([]string, error) {
			return p.storePathsFromInstallables(ctx)
		},
	}
	for _, p := range pkgs {
		paths, err := p.GetResolvedStorePaths()
		if err != nil {
			return nil, err
		}
		if len(paths) > 0 {
			q.resolved[p] = paths
			continue
		}
		if featureflag.TidyWarning.Enabled() && p.IsDevboxPackage {
			// No fast path, we need to query nix.
			ux.FHidableWarning(ctx, w, MissingStorePathsWarning, p.Raw)
		}
		installables, err := p.Installables()
		if err != nil {
			return nil, err
		}
		q.installables[p] = installables
		if p.HasAllowInsecure() {
			q.insecure = append(q.insecure, p)
		} else {
			q.secure = append(q.secure, p)
		}
	}
	return q.notInStore(ctx, pkgs)
}

// storePathQuery figures out which items (packages) are missing store paths.
// It's generic over the item type so the batching logic can be tested
// without nix.
type storePathQuery[K comparable] struct {
	// resolved holds items whose store paths are already known.
	resolved map[K][]string

	// secure and insecure hold the remaining items, split by whether nix
	// should allow insecure packages when evaluating them.
	secure, insecure []K

	// installables holds the installables of each item in secure/insecure.
	installables map[K][]string

	// pathInfo returns the union of store paths for some installables.
	pathInfo func(ctx context.Context, installables []string, allowInsecure bool) ([]string, error)

	// inStore reports which store paths are in the nix store.
	inStore func(ctx context.Context, paths []string) (map[string]bool, error)

	// single returns the store paths for one item, attributing any error to
	// that item.
	single func(ctx context.Context, item K) ([]string, error)
}

func (q *storePathQuery[K]) notInStore(ctx context.Context, items []K) ([]K, error) {
	// `nix path-info` doesn't say which installable produced which path, so
	// a batch result can't be split back up per item. That's fine as long as
	// every path is in the store: then none of the batch's items need to be
	// installed. A batch that fails or has a missing path gets retried one
	// item at a time so the error or missing path is attributed correctly.
	type batch struct {
		items []K
		paths []string
		ok    bool
	}
	batches := []*batch{}
	allPaths := []string{}
	for _, b := range q.resolved {
		allPaths = append(allPaths, b...)
	}
	for _, group := range []struct {
		items         []K
		allowInsecure bool
	}{{q.secure, false}, {q.insecure, true}} {
		if len(group.items) == 0 {
			continue
		}
		installables := []string{}
		for _, item := range group.items {
			installables = append(installables, q.installables[item]...)
		}
		b := &batch{items: group.items}
		batches = append(batches, b)

		// A single item doesn't need a batch call; retrying it would just
		// repeat the same command.
		if len(group.items) == 1 {
			continue
		}
		paths, err := q.pathInfo(ctx, installables, group.allowInsecure)
		if err != nil {
			continue
		}
		b.paths, b.ok = paths, true
		allPaths = append(allPaths, paths...)
	}

	// Individually query items from batches that failed or were skipped.
	itemPaths := map[K][]string{}
	for _, b := range batches {
		if b.ok {
			continue
		}
		if err := q.querySingles(ctx, b.items, itemPaths, &allPaths); err != nil {
			return nil, err
		}
	}

	inStore, err := q.inStore(ctx, lo.Uniq(allPaths))
	if err != nil {
		return nil, err
	}

	// Batches with a missing path need to be split up to find out which
	// items it belongs to.
	for _, b := range batches {
		if b.ok && !allInStore(b.paths, inStore) {
			newPaths := []string{}
			if err := q.querySingles(ctx, b.items, itemPaths, &newPaths); err != nil {
				return nil, err
			}
			// Single-item paths are a subset of the batch's paths, so
			// we already know whether they're in the store. Query any
			// stragglers just in case.
			unknown := []string{}
			for _, p := range newPaths {
				if _, ok := inStore[p]; !ok {
					unknown = append(unknown, p)
				}
			}
			if len(unknown) > 0 {
				more, err := q.inStore(ctx, unknown)
				if err != nil {
					return nil, err
				}
				for p, v := range more {
					inStore[p] = v
				}
			}
		}
	}

	missing := []K{}
	for _, item := range items {
		paths, ok := q.resolved[item]
		if !ok {
			paths, ok = itemPaths[item]
		}
		if !ok {
			// Part of a batch where everything is in the store.
			continue
		}
		if !allInStore(paths, inStore) {
			missing = append(missing, item)
		}
	}
	return missing, nil
}

func (q *storePathQuery[K]) querySingles(ctx context.Context, items []K, itemPaths map[K][]string, allPaths *[]string) error {
	for _, item := range items {
		paths, err := q.single(ctx, item)
		if err != nil {
			return err
		}
		itemPaths[item] = paths
		*allPaths = append(*allPaths, paths...)
	}
	return nil
}

func allInStore(paths []string, inStore map[string]bool) bool {
	for _, p := range paths {
		if !inStore[p] {
			return false
		}
	}
	return true
}
