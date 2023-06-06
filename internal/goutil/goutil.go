// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package goutil

import "github.com/samber/lo"

func PickByKeysSorted[K comparable, V any](in map[K]V, keys []K) []V {
	out := make([]V, len(keys))
	for i, key := range keys {
		out[i] = in[key]
	}
	return out
}

func Except[T comparable](in []T, except ...T) []T {
	out := []T{}
	for _, item := range in {
		if !lo.Contains(except, item) {
			out = append(out, item)
		}
	}
	return out
}
