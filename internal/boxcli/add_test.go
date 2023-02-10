// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package boxcli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.jetpack.io/devbox/internal/testframework"
)

func TestAdd(t *testing.T) {
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
	output, err := td.RunCommand(AddCmd(), "hello")
	assert.NoError(t, err)
	assert.Contains(t, output, "hello (hello-2.12.1) is now installed.")
	updatedDevboxJSON, err := td.GetDevboxJSON()
	assert.NoError(t, err)
	assert.Contains(t, updatedDevboxJSON.RawPackages, "hello")
}
