package configfile

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeprecatedShellBackwardCompat verifies that the legacy nested "shell"
// object is still readable via the InitHook and Scripts accessors.
func TestDeprecatedShellBackwardCompat(t *testing.T) {
	const json = `{
		"shell": {
			"init_hook": ["echo hi"],
			"scripts": {
				"build": "go build ./..."
			}
		}
	}`

	cfg, err := LoadBytes([]byte(json))
	require.NoError(t, err)

	assert.True(t, cfg.UsesDeprecatedShellField())
	assert.Equal(t, []string{"echo hi"}, cfg.InitHook().Cmds)

	scripts := cfg.Scripts()
	require.Contains(t, scripts, "build")
	assert.Equal(t, []string{"go build ./..."}, scripts["build"].Cmds)
}

// TestTopLevelShellFields verifies that the modern top-level init_hook and
// scripts fields are read.
func TestTopLevelShellFields(t *testing.T) {
	const json = `{
		"init_hook": ["echo hi"],
		"scripts": {
			"build": "go build ./..."
		}
	}`

	cfg, err := LoadBytes([]byte(json))
	require.NoError(t, err)

	assert.False(t, cfg.UsesDeprecatedShellField())
	assert.Equal(t, []string{"echo hi"}, cfg.InitHook().Cmds)

	scripts := cfg.Scripts()
	require.Contains(t, scripts, "build")
	assert.Equal(t, []string{"go build ./..."}, scripts["build"].Cmds)
}

// TestMigrateShell verifies that MigrateShell moves the nested shell fields to
// the top level, removes the "shell" object, and preserves the data.
func TestMigrateShell(t *testing.T) {
	const json = `{
		"packages": ["go@latest"],
		"shell": {
			"init_hook": ["echo hi"],
			"scripts": {
				"build": "go build ./..."
			}
		}
	}`

	cfg, err := LoadBytes([]byte(json))
	require.NoError(t, err)

	cfg.MigrateShell()

	// The struct no longer references the deprecated shell object.
	assert.False(t, cfg.UsesDeprecatedShellField())
	assert.Equal(t, []string{"echo hi"}, cfg.InitHook().Cmds)
	require.Contains(t, cfg.Scripts(), "build")

	// The serialized output should no longer contain a "shell" object, but
	// should have top-level init_hook and scripts.
	out := string(cfg.Bytes())
	assert.NotContains(t, out, `"shell"`)
	assert.Contains(t, out, `"init_hook"`)
	assert.Contains(t, out, `"scripts"`)

	// The migrated config should round-trip and still parse.
	reparsed, err := LoadBytes(cfg.Bytes())
	require.NoError(t, err)
	assert.False(t, reparsed.UsesDeprecatedShellField())
	assert.Equal(t, []string{"echo hi"}, reparsed.InitHook().Cmds)
	require.Contains(t, reparsed.Scripts(), "build")
}

// TestMigrateShellPreservesComments verifies comments survive migration.
func TestMigrateShellPreservesComments(t *testing.T) {
	const json = `{
		"shell": {
			// build the project
			"scripts": {
				"build": "go build ./..."
			}
		}
	}`

	cfg, err := LoadBytes([]byte(json))
	require.NoError(t, err)
	cfg.MigrateShell()

	out := string(cfg.Bytes())
	assert.True(t, strings.Contains(out, "build the project"),
		"expected comment to be preserved, got:\n%s", out)
}

// TestMigrateShellNoShell is a no-op when there's no shell object.
func TestMigrateShellNoShell(t *testing.T) {
	cfg, err := LoadBytes([]byte(`{"packages": []}`))
	require.NoError(t, err)
	require.NotPanics(t, func() { cfg.MigrateShell() })
	assert.False(t, cfg.UsesDeprecatedShellField())
}
