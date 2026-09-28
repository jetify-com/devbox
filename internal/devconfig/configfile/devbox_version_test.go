package configfile

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDevboxVersionShorthand(t *testing.T) {
	cfg, err := LoadBytes([]byte(`{"devbox_version": "^0.18.0"}`))
	require.NoError(t, err)
	require.NotNil(t, cfg.DevboxVersion)
	assert.Equal(t, "^0.18.0", cfg.DevboxVersion.Version)
	assert.Equal(t, VersionPolicyWarn, cfg.DevboxVersion.Policy())

	b, err := json.Marshal(cfg.DevboxVersion)
	require.NoError(t, err)
	assert.JSONEq(t, `"^0.18.0"`, string(b))
}

func TestDevboxVersionObject(t *testing.T) {
	cfg, err := LoadBytes([]byte(`{
		"devbox_version": {"version": "0.18.4", "on_mismatch": "error"}
	}`))
	require.NoError(t, err)
	require.NotNil(t, cfg.DevboxVersion)
	assert.Equal(t, "0.18.4", cfg.DevboxVersion.Version)
	assert.Equal(t, VersionPolicyError, cfg.DevboxVersion.Policy())

	b, err := json.Marshal(cfg.DevboxVersion)
	require.NoError(t, err)
	assert.JSONEq(t, `{"version": "0.18.4", "on_mismatch": "error"}`, string(b))
}

func TestDevboxVersionObjectDefaultsToWarn(t *testing.T) {
	cfg, err := LoadBytes([]byte(`{"devbox_version": {"version": "0.18.4"}}`))
	require.NoError(t, err)
	assert.Equal(t, VersionPolicyWarn, cfg.DevboxVersion.Policy())
}

func TestDevboxVersionUnset(t *testing.T) {
	cfg, err := LoadBytes([]byte(`{"packages": []}`))
	require.NoError(t, err)
	assert.Nil(t, cfg.DevboxVersion)
}

func TestDevboxVersionValidation(t *testing.T) {
	valid := []string{
		`"0.18.4"`,
		`"v0.18.4"`,
		`"^0.18.0"`,
		`"~0.18.1"`,
		`">=0.17.0 <0.19.0"`,
		`">=0.17.0, <0.19.0"`,
		`"0.18.x"`,
		`"0.17.2 || ^0.18.0"`,
		`{"version": "0.18.4", "on_mismatch": "warn"}`,
	}
	for _, v := range valid {
		t.Run(v, func(t *testing.T) {
			_, err := LoadBytes([]byte(`{"devbox_version": ` + v + `}`))
			assert.NoError(t, err)
		})
	}

	invalid := []string{
		`""`,
		`"latest"`,
		`">=> 1"`,
		`{}`,
		`{"on_mismatch": "error"}`,
		`{"version": "0.18.4", "on_mismatch": "explode"}`,
		`{"version": "0.18.4", "on_mismatch": "off"}`,
	}
	for _, v := range invalid {
		t.Run(v, func(t *testing.T) {
			_, err := LoadBytes([]byte(`{"devbox_version": ` + v + `}`))
			assert.Error(t, err)
		})
	}
}

func TestDevboxVersionExactVersion(t *testing.T) {
	tests := map[string]struct {
		want string
		ok   bool
	}{
		"0.18.4":           {"0.18.4", true},
		"v0.18.4":          {"0.18.4", true},
		" 0.18.4 ":         {"0.18.4", true},
		"0.18.4-rc1":       {"0.18.4-rc1", true},
		"0.18":             {"", false},
		"^0.18.0":          {"", false},
		"=0.18.4":          {"", false},
		">=0.17.0 <0.19.0": {"", false},
	}
	for version, tt := range tests {
		t.Run(version, func(t *testing.T) {
			got, ok := (&DevboxVersion{Version: version}).ExactVersion()
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseVersionPolicy(t *testing.T) {
	for _, s := range []string{"off", "warn", "error", " WARN "} {
		_, err := ParseVersionPolicy(s)
		assert.NoError(t, err, s)
	}
	_, err := ParseVersionPolicy("explode")
	assert.Error(t, err)
}
