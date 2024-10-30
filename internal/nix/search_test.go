package nix

import (
	"reflect"
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
		{
			"github:nixos/nixpkgs/7d0ed7f2e5aea07ab22ccb338d27fbe347ed2f11#emacsPackages.@",
			"github_nixos_nixpkgs_7d0ed7f2e5aea07ab22ccb338d27fbe347ed2f11_emacsPackages._",
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

func TestAllowableQuery(t *testing.T) {
	testCases := []struct {
		in       string
		expected bool
	}{
		{
			"github:NixOS/nixpkgs/8670e496ffd093b60e74e7fa53526aa5920d09eb#go_1_19",
			true,
		},
		{
			"github:NixOS/nixpkgs/8670e496ffd093b60e74e7fa53526aa5920d09eb",
			false,
		},
		{
			"github:NixOS/nixpkgs/8670e496ffd093b60e74e7fa53526aa5920d09eb#",
			false,
		},
		{
			"github:NixOS/nixpkgs/nixpkgs-unstable#go_1_19",
			false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.in, func(t *testing.T) {
			out := allowableQuery.MatchString(testCase.in)
			if out != testCase.expected {
				t.Errorf("got %t, want %t", out, testCase.expected)
			}
		})
	}
}

func TestParseSearchResults(t *testing.T) {
	testCases := []struct {
		name           string
		input          []byte
		expectedResult map[string]*Info
	}{
		{
			name: "Valid JSON input",
			input: []byte(`{
				"go": {
					"pname": "go",
					"version": "1.20.4"
				},
				"python3": {
					"pname": "python3",
					"version": "3.9.16"
				}
			}`),
			expectedResult: map[string]*Info{
				"go": {
					AttributeKey: "go",
					PName:        "go",
					Version:      "1.20.4",
				},
				"python3": {
					AttributeKey: "python3",
					PName:        "python3",
					Version:      "3.9.16",
				},
			},
		},
		{
			name:           "Empty JSON input",
			input:          []byte(`{}`),
			expectedResult: map[string]*Info{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseSearchResults(tc.input)

			if !reflect.DeepEqual(result, tc.expectedResult) {
				t.Errorf("Expected result %v, got %v", tc.expectedResult, result)
			}
		})
	}
}
