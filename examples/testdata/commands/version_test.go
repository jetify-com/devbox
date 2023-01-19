package testcommands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.jetpack.io/devbox/examples/testdata/testframework"
)

func TestVersion(t *testing.T) {
	td := testframework.Open()
	output, err := td.Version()
	assert.NoError(t, err)
	assert.Contains(t, output, "0.2.2")
}
