package cuecfg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type Metadata struct {
	Tags []string
}

type MyConfig struct {
	Version int
	Name    string
	Meta    *Metadata
}

var testTomlCfg = &MyConfig{
	Version: 2,
	Name:    "go-toml",
	Meta: &Metadata{
		Tags: []string{"go", "toml"},
	},
}

var testTomlStr = `Version = 2
Name = 'go-toml'

[Meta]
Tags = ['go', 'toml']
`

func TestMarshalToml(t *testing.T) {
	req := require.New(t)

	bytes, err := Marshal(testTomlCfg, ".toml")
	req.NoError(err)

	req.Equal(testTomlStr, string(bytes))
}

func TestUnmarshalToml(t *testing.T) {
	req := require.New(t)
	cfg := &MyConfig{}
	err := Unmarshal([]byte(testTomlStr), ".toml", cfg)
	req.NoError(err)
	req.Equal(testTomlCfg, cfg)
}
