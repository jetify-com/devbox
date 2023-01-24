// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package testrun

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.jetpack.io/devbox/examples/testdata/testframework"
)

func TestRun(t *testing.T) {
	td := testframework.Open()
	err := td.SetDevboxJson("devbox.json")
	assert.NoError(t, err)
	_, err = td.Run("test1")
	assert.NoError(t, err)
}
