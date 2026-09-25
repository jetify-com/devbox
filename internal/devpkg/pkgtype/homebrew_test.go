package pkgtype

import "testing"

func TestIsHomebrew(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"homebrew:python@3.10", true},
		{"homebrew:wget", true},
		{"homebrew:", true},
		{"runx:golangci/golangci-lint", false},
		{"python@3.10", false},
		{"github:NixOS/nixpkgs/12345", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := IsHomebrew(tt.in); got != tt.want {
				t.Errorf("IsHomebrew(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestHomebrewIsNotFlake(t *testing.T) {
	for _, s := range []string{"homebrew:python@3.10", "homebrew:wget"} {
		if IsFlake(s) {
			t.Errorf("IsFlake(%q) = true, want false", s)
		}
	}
}
