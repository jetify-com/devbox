// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package goutil

func PickByKeysSorted[K comparable, V any](in map[K]V, keys []K) []V {
	out := make([]V, len(keys))
	for i, key := range keys {
		out[i] = in[key]
	}
	return out
}
