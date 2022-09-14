package plansdk

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionExact(t *testing.T) {
	cases := []struct {
		version string
		exact   string
	}{
		{"1", "1"},
		{"1.2", "1.2"},
		{"1.2.3", "1.2.3"},
	}

	for _, tc := range cases {
		t.Run(
			tc.version, func(t *testing.T) {
				req := require.New(t)

				v, err := NewVersion(tc.version)
				req.NoError(err)
				req.NotNil(v)
				req.Equal(tc.exact, v.Exact())
			},
		)
	}
}
