// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
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

func TestGetTemplateRepoAndSubdir(t *testing.T) {
	// devbox create --template=nonexistenttemplate
	_, _, err := GetTemplateRepoAndSubdir(
		"nonexistenttemplate",
		"",
		"",
	)
	assert.Error(t, err)

	// devbox create --template=apache --repo="https://example.com/org/repo.git" \
	// --subdir="examples/test/should/ignore/repo/and/subdir"
	repoURL, subdirPath, err := GetTemplateRepoAndSubdir(
		"apache",
		"https://example.com/org/repo.git",
		"examples/test/should/ignore/repo/and/subdir",
	)
	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/jetpack-io/devbox", repoURL)
	assert.Equal(t, "examples/servers/apache/", subdirPath)

	// devbox create --repo="https://github.com/jetpack-io/typeid.git"
	repoURL, subdirPath, err = GetTemplateRepoAndSubdir(
		"",
		"https://github.com/jetpack-io/typeid.git",
		"",
	)
	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/jetpack-io/typeid", repoURL)
	assert.Equal(t, "", subdirPath)

	// devbox create --repo="https://github.com/jetpack-io/devbox" \
	// --subdir="examples/servers/apache"
	repoURL, subdirPath, err = GetTemplateRepoAndSubdir(
		"",
		"https://github.com/jetpack-io/devbox",
		"examples/servers/apache",
	)
	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/jetpack-io/devbox", repoURL)
	assert.Equal(t, "examples/servers/apache", subdirPath)

	// devbox create --repo="git@github.com:jetpack-io/devbox.git" \
	// --subdir="examples/servers/apache"
	repoURL, subdirPath, err = GetTemplateRepoAndSubdir(
		"",
		"git@github.com:jetpack-io/devbox.git",
		"examples/servers/apache",
	)
	assert.NoError(t, err)
	assert.Equal(t, "git@github.com:jetpack-io/devbox", repoURL)
	assert.Equal(t, "examples/servers/apache", subdirPath)
}
