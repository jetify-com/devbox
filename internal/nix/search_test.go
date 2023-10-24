package nix

import (
	"testing"
)

func TestSearchCacheKey(t *testing.T) {
	testCases := []struct {
		in  string
		out string
	}{
		{
			"github:NixOS/nixpkgs/8670e496ffd093b60e74e7fa53526aa5920d09eb#go_1_19",
			"github_NixOS_nixpkgs_8670e496ffd093b60e74e7fa53526aa5920d09eb_go_1_19",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.out, func(t *testing.T) {
			out := cacheKey(testCase.in)
			if out != testCase.out {
				t.Errorf("got %s, want %s", out, testCase.out)
			}
		})
	}
}
