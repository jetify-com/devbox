package cuecfg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type MyConfig struct {
	Version int
	Name    string
	Tags    []string
}

var testTomlCfg = &MyConfig{
	Version: 2,
	Name:    "go-toml",
	Tags:    []string{"go", "toml"},
}

func TestMarshalToml(t *testing.T) {
	req := require.New(t)

	bytes, err := Marshal(testTomlCfg, ".toml")
	req.NoError(err)

	expected := `Version = 2
Name = 'go-toml'
Tags = ['go', 'toml']
`
	req.Equal(expected, string(bytes))
}

func TestUnmarshalToml(t *testing.T) {
	req := require.New(t)
	tomlStr := `Version = 2
Name = 'go-toml'
Tags = ['go', 'toml']
`
	cfg := &MyConfig{}
	err := Unmarshal([]byte(tomlStr), ".toml", cfg)
	req.NoError(err)
	req.Equal(testTomlCfg, cfg)
}
