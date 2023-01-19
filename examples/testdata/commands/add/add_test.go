package testadd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.jetpack.io/devbox/examples/testdata/testframework"
)

func TestAdd(t *testing.T) {
	td := testframework.Open()
	output, err := td.Add("go_1_17")
	assert.NoError(t, err)
	assert.Contains(t, output, "go_1_17 (go-1.17.13) is now installed.")
	td.SetDevboxJson("devbox.json")
	devboxjson, _ := td.GetDevboxJson()
	assert.Contains(t, devboxjson.Packages, "go_1_17")
}
