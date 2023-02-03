// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package boxcli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.jetpack.io/devbox/internal/testframework"
)

func TestInit(t *testing.T) {
	td := testframework.Open()
	defer td.Close()
	_, err := td.RunCommand(InitCmd())
	assert.NoError(t, err)
	assert.FileExists(t, "devbox.json")
}

func TestInitRecommendation(t *testing.T) {
	td := testframework.Open()
	defer td.Close()
	err := td.CreateFile("package.json", "{}")
	assert.NoError(t, err)
	err = td.CreateFile("requirements.txt", "")
	assert.NoError(t, err)
	output, err := td.RunCommand(InitCmd())
	assert.NoError(t, err)
	assert.FileExists(t, "devbox.json")
	assert.Contains(t, output, "We detected extra packages you may need.")
	assert.Contains(t, output, "devbox add")
	assert.Contains(t, output, "nodejs")
	assert.Contains(t, output, "python3")
}
