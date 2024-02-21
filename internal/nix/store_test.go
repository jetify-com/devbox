package nix

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseStorePathFromInstallableOutput(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "go-basic",
			// snipped the actual output for brevity. We mainly care about the first key in the JSON.
			input:    `{"/nix/store/fgkl3qk8p5hnd07b0dhzfky3ys5gxjmq-go-1.22.0":{"deriver":"/nix/store/clr3bm8njqysvyw4r4x4xmldhz4knrff-go-1.22.0.drv"}}`,
			expected: "/nix/store/fgkl3qk8p5hnd07b0dhzfky3ys5gxjmq-go-1.22.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := parseStorePathFromInstallableOutput(tc.name, []byte(tc.input))
			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}
