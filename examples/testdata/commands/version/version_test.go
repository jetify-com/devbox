// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package testversion

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.jetpack.io/devbox/examples/testdata/testframework"
)

func TestVersion(t *testing.T) {
	td := testframework.Open()
	output, err := td.Version()
	assert.NoError(t, err)
	assert.Contains(t, output, "0.0.0-dev")
}
