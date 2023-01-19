package testshell

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.jetpack.io/devbox/examples/testdata/testframework"
)

func TestShell(t *testing.T) {
	td := testframework.Open()
	err := td.SetDevboxJson("devbox.json")
	assert.NoError(t, err)
	output, err := td.Shell()
	assert.NoError(t, err)
	assert.Contains(t, output, "Starting a devbox shell...")
}
