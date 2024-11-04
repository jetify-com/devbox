package configfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tailscale/hujson"
)

func TestSetEnv(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		env      map[string]string
		expected string
	}{
		{
			name:    "add env to empty config",
			initial: "{}",
			env: map[string]string{
				"FOO": "bar",
				"BAZ": "qux",
			},
			expected: `{"env": {"FOO": "bar", "BAZ": "qux"}}
`,
		},
		{
			name: "update existing env",
			initial: `{
	"env": {
		"EXISTING": "value"
	}
}`,
			env: map[string]string{
				"FOO": "bar",
			},
			expected: `{
	"env": {"FOO": "bar"}
}
`,
		},
		{
			name: "clear env with empty map",
			initial: `{
	"env": {
		"EXISTING": "value"
	}
}`,
			env: map[string]string{},
			expected: `{
	"env": {}
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := hujson.Parse([]byte(tt.initial))
			assert.NoError(t, err)

			ast := &configAST{root: val}
			ast.setEnv(tt.env)

			actual := string(ast.root.Pack())
			assert.Equal(t, tt.expected, actual)
		})
	}
}
