// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package boxcli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.jetpack.io/devbox/internal/testframework"
)

func TestGenerateDockerfile(t *testing.T) {
	devboxJSON := `
	{
		"packages": [],
		"shell": {
		  "init_hook": null
		},
		"nixpkgs": {
		  "commit": "af9e00071d0971eb292fd5abef334e66eda3cb69"
		}
	}`
	td := testframework.Open()
	defer td.Close()
	err := td.SetDevboxJSON(devboxJSON)
	assert.NoError(t, err)
	// devbox generate dockerfile doesn't generate any output
	_, err = td.RunCommand(GenerateCmd(), "dockerfile")
	assert.NoError(t, err)
	assert.FileExists(t, "Dockerfile")
}

func TestGenerateDevcontainer(t *testing.T) {
	devboxJSON := `
	{
		"packages": [],
		"shell": {
		  "init_hook": null
		},
		"nixpkgs": {
		  "commit": "af9e00071d0971eb292fd5abef334e66eda3cb69"
		}
	}`
	td := testframework.Open()
	defer td.Close()
	err := td.SetDevboxJSON(devboxJSON)
	assert.NoError(t, err)
	// devbox generate devcontainer doesn't generate any output
	_, err = td.RunCommand(GenerateCmd(), "devcontainer")
	assert.NoError(t, err)
	assert.FileExists(t, ".devcontainer/Dockerfile")
	assert.FileExists(t, ".devcontainer/devcontainer.json")
}
