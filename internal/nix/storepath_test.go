package nix

import (
	"testing"
)

func TestStorePathParts(t *testing.T) {
	testCases := []struct {
		storePath string
		expected  StorePathParts
	}{
		// simple case:
		{
			storePath: "/nix/store/cvrn84c1hshv2wcds7n1rhydi6lacqns-gnumake-4.4.1",
			expected: StorePathParts{
				Hash:    "cvrn84c1hshv2wcds7n1rhydi6lacqns",
				Name:    "gnumake",
				Version: "4.4.1",
			},
		},
		// the package name can have dashes:
		{
			storePath: "/nix/store/q2xdxsswjqmqcbax81pmazm367s7jzyb-cctools-binutils-darwin-wrapper-973.0.1",
			expected: StorePathParts{
				Hash:    "q2xdxsswjqmqcbax81pmazm367s7jzyb",
				Name:    "cctools-binutils-darwin-wrapper",
				Version: "973.0.1",
			},
		},
		// version is optional. This is an artificial example I constructed
		{
			storePath: "/nix/store/gfxwrd5nggc68pjj3g3jhlldim9rpg0p-coreutils",
			expected: StorePathParts{
				Hash: "gfxwrd5nggc68pjj3g3jhlldim9rpg0p",
				Name: "coreutils",
			},
		},
		// With output
		{
			storePath: "/nix/store/0z1zq1zq1zq1zq1zq1zq1zq1zq1zq1zq-foo-1.0.0-bar",
			expected: StorePathParts{
				Hash:    "0z1zq1zq1zq1zq1zq1zq1zq1zq1zq1zq",
				Name:    "foo",
				Version: "1.0.0",
				Output:  "bar",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.storePath, func(t *testing.T) {
			parts := NewStorePathParts(testCase.storePath)
			if parts != testCase.expected {
				t.Errorf("Expected %v, got %v", testCase.expected, parts)
			}
		})
	}
}
