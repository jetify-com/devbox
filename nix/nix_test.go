package nix

import "testing"

func TestPkgExists(t *testing.T) {
	// nix-env returns an empty JSON object instead of an error for some
	// missing packages, which was leading to a panic. "rust" happens to be
	// one of those packages.
	pkg := "rust"
	exists := PkgExists(pkg)
	if exists {
		t.Errorf("got PkgExists(%q) = true, want false.", pkg)
	}
}
