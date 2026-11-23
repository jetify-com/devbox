package pkgtype

import (
	"strings"
	"testing"
)

func TestCheckPkgDefEnabled(t *testing.T) {
	t.Setenv("DEVBOX_FEATURE_PKG_DEFS", "0")
	err := CheckPkgDefEnabled("pkg:./x.pkg.jsonc")
	if err == nil || !strings.Contains(err.Error(), "DEVBOX_FEATURE_PKG_DEFS=1") {
		t.Errorf("flag off: got %v, want an error naming DEVBOX_FEATURE_PKG_DEFS=1", err)
	}
	if err := CheckPkgDefEnabled("hello@1.2.3"); err != nil {
		t.Errorf("flag off, nix package: got %v, want nil", err)
	}

	t.Setenv("DEVBOX_FEATURE_PKG_DEFS", "1")
	if err := CheckPkgDefEnabled("pkg:./x.pkg.jsonc"); err != nil {
		t.Errorf("flag on: got %v, want nil", err)
	}
}
