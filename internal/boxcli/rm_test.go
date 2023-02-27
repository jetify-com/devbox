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
			"hello"
		],
		"shell": {
		  "scripts": {
			"test1": "echo test1"
		  },
		  "init_hook": null
		},
		"nixpkgs": {
		  "commit": "af9e00071d0971eb292fd5abef334e66eda3cb69"
		}
	}`
	testbox := testframework.Open()
	defer testbox.Close()
	err := testbox.SetDevboxJSON(devboxJSON)
	assert.NoError(t, err)

	// First, run a devbox script to install the packages as a side-effect
	_, err = testbox.RunCommand(RunCmd(), "test1")
	assert.NoError(t, err)

	// Now, run the Remove command
	output, err := testbox.RunCommand(RemoveCmd(), "hello")
	assert.NoError(t, err)
	assert.Contains(t, output, "hello (hello-2.12.1) is now removed.")
	devboxjson, err := testbox.GetDevboxJSON()
	assert.NoError(t, err)
	assert.NotContains(t, devboxjson.RawPackages, "hello")
}
