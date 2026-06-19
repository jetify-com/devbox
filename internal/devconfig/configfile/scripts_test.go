package configfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scriptNames returns just the names of the ordered scripts, for convenient
// assertions.
func scriptNames(scripts []ScriptWithName) []string {
	names := make([]string, 0, len(scripts))
	for _, s := range scripts {
		names = append(names, s.Name)
	}
	return names
}

func TestScriptsInOrderPreservesSourceOrder(t *testing.T) {
	config := []byte(`{
		"shell": {
			"scripts": {
				"step-one": "echo one",
				"step-two": "echo two",
				"step-three": "echo three",
				"step-four": "echo four"
			}
		}
	}`)

	cfg, err := LoadBytes(config)
	require.NoError(t, err)

	ordered := cfg.Scripts().InOrder(cfg.ScriptOrder())
	assert.Equal(t,
		[]string{"step-one", "step-two", "step-three", "step-four"},
		scriptNames(ordered),
	)
}

func TestScriptsInOrderFallsBackToAlphabetical(t *testing.T) {
	// When no order is provided (e.g. the config wasn't parsed from a file),
	// the result should be deterministic (alphabetical) rather than random
	// map order.
	scripts := Scripts{
		"build": &script{},
		"test":  &script{},
		"clean": &script{},
	}

	ordered := scripts.InOrder(nil)
	assert.Equal(t, []string{"build", "clean", "test"}, scriptNames(ordered))
}

func TestScriptsInOrderAppendsUnknownScripts(t *testing.T) {
	// Scripts present in the map but missing from the order slice should be
	// appended in alphabetical order, with no duplicates.
	scripts := Scripts{
		"first":  &script{},
		"second": &script{},
		"zeta":   &script{},
		"alpha":  &script{},
	}

	ordered := scripts.InOrder([]string{"first", "second"})
	assert.Equal(t,
		[]string{"first", "second", "alpha", "zeta"},
		scriptNames(ordered),
	)
}

func TestScriptsInOrderCarriesCommands(t *testing.T) {
	config := []byte(`{
		"shell": {
			"scripts": {
				"greet": ["echo hello", "echo world"]
			}
		}
	}`)

	cfg, err := LoadBytes(config)
	require.NoError(t, err)

	ordered := cfg.Scripts().InOrder(cfg.ScriptOrder())
	require.Len(t, ordered, 1)
	assert.Equal(t, "greet", ordered[0].Name)
	require.NotNil(t, ordered[0].Commands)
	assert.Equal(t, "echo hello\necho world", ordered[0].Commands.String())
}
