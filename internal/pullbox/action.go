// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pullbox

type Action int

const (
	NoAction = iota
	MergeAction
	OverwriteAction
)

func ActionFromString(s string) Action {
	switch s {
	case "merge":
		return MergeAction
	case "overwrite":
		return OverwriteAction
	default:
		return NoAction
	}
}
