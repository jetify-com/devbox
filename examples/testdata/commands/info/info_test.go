// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package testinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.jetpack.io/devbox/examples/testdata/testframework"
)

func TestInfo(t *testing.T) {
	td := testframework.Open()
	output, err := td.Info("notarealpackage", false)
	assert.NoError(t, err)
	assert.Contains(t, output, "Package notarealpackage not found")
}
