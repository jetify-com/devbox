// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDevboxBinaryForSelfInvocation verifies that `devbox services ...`
// re-invokes devbox using the path to the currently running binary rather than
// a hardcoded "devbox" that must be resolvable on PATH. This is a regression
// guard for https://github.com/jetify-com/devbox/issues/1321, where services
// commands broke whenever the binary was installed or renamed to something
// other than "devbox".
func TestDevboxBinaryForSelfInvocation(t *testing.T) {
	exe, err := os.Executable()
	require.NoError(t, err)

	got := devboxBinaryForSelfInvocation()

	// It should reference the actual running binary (shell-quoted), not the
	// literal command name "devbox".
	assert.Equal(t, strconv.Quote(exe), got)
	assert.NotEqual(t, "devbox", got)
	assert.NotEqual(t, strconv.Quote("devbox"), got)
}
