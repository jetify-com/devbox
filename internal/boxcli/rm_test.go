// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package boxcli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.jetpack.io/devbox/internal/testframework"
)

func TestRm(t *testing.T) {
	devboxJSON := `
	{
		"packages": [
			"go_1_17"
		],
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
	output, err := td.RunCommand(RemoveCmd(), "go_1_17")
	assert.NoError(t, err)
	assert.Contains(t, output, "go_1_17 (go-1.17.13) is now removed.")
	devboxjson, err := td.GetDevboxJSON()
	assert.NoError(t, err)
	assert.NotContains(t, devboxjson.Packages, "go_1_17")
}
