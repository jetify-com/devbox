// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package testgenrate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.jetpack.io/devbox/examples/testdata/testframework"
)

func TestGenerateDockerfile(t *testing.T) {
	td := testframework.Open()
	err := td.SetDevboxJson("devbox.json")
	assert.NoError(t, err)
	// devbox generate dockerfile doesn't generate any output
	_, err = td.Generate("dockerfile")
	assert.NoError(t, err)
	assert.FileExists(t, "Dockerfile")
}

func TestGenerateDevcontainer(t *testing.T) {
	td := testframework.Open()
	err := td.SetDevboxJson("devbox.json")
	assert.NoError(t, err)
	// devbox generate devcontainer doesn't generate any output
	_, err = td.Generate("devcontainer")
	assert.NoError(t, err)
	assert.FileExists(t, ".devcontainer/Dockerfile")
	assert.FileExists(t, ".devcontainer/devcontainer.json")
}
