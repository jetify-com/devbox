// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package boxcli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.jetpack.io/devbox/internal/testframework"
)

func TestInfo(t *testing.T) {
	td := testframework.Open()
	defer td.Close()
	output, err := td.RunCommand(InfoCmd(), "notarealpackage")
	assert.NoError(t, err)
	assert.Contains(t, output, "Package notarealpackage not found")
}
