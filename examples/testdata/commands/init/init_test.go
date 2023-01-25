// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package testinit

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.jetpack.io/devbox/examples/testdata/testframework"
)

func TestInit(t *testing.T) {
	td := testframework.Open()
	_, err := td.Init()
	assert.NoError(t, err)
	assert.FileExists(t, "devbox.json")
}

func TestInitRecommendation(t *testing.T) {
	td := testframework.Open()
	err := exec.Command("touch", "package.json").Run()
	assert.NoError(t, err)
	err = exec.Command("touch", "requirements.txt").Run()
	assert.NoError(t, err)
	output, err := td.Init()
	assert.NoError(t, err)
	assert.FileExists(t, "devbox.json")
	assert.Contains(t, output, "We detected extra packages you may need.")
	assert.Contains(t, output, "devbox add")
	assert.Contains(t, output, "nodejs")
	assert.Contains(t, output, "python3")
}
