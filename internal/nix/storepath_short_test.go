package nix

import "testing"

func TestStorePathPartsMalformedPath(t *testing.T) {
	for _, path := range []string{
		"",
		"/nix/store/",
		"/nix/store/abc-hello-1.0",
		"/nix/store/cvrn84c1hshv2wcds7n1rhydi6lacqns",
		// Long enough, but without a dash after the hash.
		"/nix/store/cvrn84c1hshv2wcds7n1rhydi6lacqnsXhello-1.0",
	} {
		t.Run(path, func(t *testing.T) {
			if got := NewStorePathParts(path); got != (StorePathParts{}) {
				t.Errorf("NewStorePathParts(%q) = %+v, want the zero value", path, got)
			}
		})
	}
}
