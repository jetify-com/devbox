// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package testadd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.jetpack.io/devbox/examples/testdata/testframework"
)

func TestAdd(t *testing.T) {
	td := testframework.Open()
	output, err := td.Add("go_1_17")
	assert.NoError(t, err)
	assert.Contains(t, output, "go_1_17 (go-1.17.13) is now installed.")
	err = td.SetDevboxJson("devbox.json")
	assert.NoError(t, err)
	devboxjson, err := td.GetDevboxJson()
	assert.NoError(t, err)
	assert.Contains(t, devboxjson.Packages, "go_1_17")
}
