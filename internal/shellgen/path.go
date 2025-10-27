// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package shellgen

import "path/filepath"

func genPath(d devboxer) string {
	return filepath.Join(d.ProjectDir(), ".devbox/gen")
}

func FlakePath(d devboxer) string {
	return filepath.Join(genPath(d), "flake")
}
