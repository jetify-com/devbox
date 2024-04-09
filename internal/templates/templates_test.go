// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package templates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestTemplatesExist(t *testing.T) {
	curDir := ""
	// Try to find examples dir. After 10 hops, we give up.
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(curDir + "examples"); err == nil {
			break
		}
		curDir += "../"
	}
	for _, path := range templates {
		_, err := os.Stat(filepath.Join(curDir, path, "devbox.json"))
		if errors.Is(err, os.ErrNotExist) {
			t.Errorf("Directory/devbox.json for %s does not exist", path)
		}
	}
}

func TestParseRepoURL(t *testing.T) {
	// devbox create --repo="http:::/not.valid/a//a??a?b=&&c#hi"
	_, err := ParseRepoURL("http:::/not.valid/a//a??a?b=&&c#hi")
	assert.Error(t, err)
	_, err = ParseRepoURL("http//github.com")
	assert.Error(t, err)
	_, err = ParseRepoURL("github.com")
	assert.Error(t, err)
	_, err = ParseRepoURL("/foo/bar")
	assert.Error(t, err)
	_, err = ParseRepoURL("http://")
	assert.Error(t, err)
	_, err = ParseRepoURL("git@github.com:jetify-com/devbox.git")
	assert.Error(t, err)
	u, err := ParseRepoURL("http://github.com")
	assert.NoError(t, err)
	assert.Equal(t, "http://github.com", u)
}
