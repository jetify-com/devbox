package testadd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.jetpack.io/devbox/examples/testdata/testframework"
)

func TestRm(t *testing.T) {
	td := testframework.Open()
	output, err := td.Rm("go_1_17")
	assert.NoError(t, err)
	assert.Contains(t, output, "go_1_17 (go-1.17.13) is now removed.")
	td.SetDevboxJson("devbox.json")
	devboxjson, err := td.GetDevboxJson()
	assert.NoError(t, err)
	assert.NotContains(t, devboxjson.Packages, "go_1_17")
}
