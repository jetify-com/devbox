package nim

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseBinLine(t *testing.T) {
	req := require.New(t)

	cases := []struct {
		input    string
		expected []string
	}{
		{
			"bin           = @[\"hello_world\"]",
			[]string{"hello_world"},
		},
		{
			"bin           = @[\"hello_world\", \"second_world\"]",
			[]string{"hello_world", "second_world"},
		},
	}

	for idx, tc := range cases {
		t.Run(
			fmt.Sprintf("testParseBinLine_%d", idx),
			func(t *testing.T) {
				result := parseBinLine(tc.input)
				req.Equal(tc.expected, result)
			},
		)
	}
}

func TestParseBinDirLine(t *testing.T) {
	req := require.New(t)

	cases := []struct {
		input    string
		expected string
	}{
		{
			"binDir           = \"binaries\"",
			"binaries",
		},
	}

	for idx, tc := range cases {
		t.Run(
			fmt.Sprintf("testParseBinDirLine_%d", idx),
			func(t *testing.T) {
				result := parseBinDirLine(tc.input)
				req.Equal(tc.expected, result)
			},
		)
	}
}
