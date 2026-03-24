package pkgtype

import "testing"

func TestIsJSPM(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"pnpm:vercel", true},
		{"pnpm:vercel@latest", true},
		{"yarn:turbo", true},
		{"yarn:turbo@1.0.0", true},
		{"npm:eslint", true},
		{"npm:@scope/pkg@1.0.0", true},
		{"go@1.21", false},
		{"runx:foo/bar", false},
		{"hello", false},
		{"github:NixOS/nixpkgs#hello", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsJSPM(tt.input)
			if got != tt.expected {
				t.Errorf("IsJSPM(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestJSPMType(t *testing.T) {
	tests := []struct {
		input    string
		expected JSPackageManager
	}{
		{"pnpm:vercel", Pnpm},
		{"pnpm:vercel@latest", Pnpm},
		{"yarn:turbo", Yarn},
		{"npm:eslint", Npm},
		{"npm:@scope/pkg@1.0.0", Npm},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := JSPMType(tt.input)
			if got != tt.expected {
				t.Errorf("JSPMType(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestJSPMPackageName(t *testing.T) {
	tests := []struct {
		input        string
		expectedName string
		expectedVer  string
	}{
		{"pnpm:vercel@latest", "vercel", "latest"},
		{"pnpm:vercel", "vercel", ""},
		{"pnpm:vercel@1.2.3", "vercel", "1.2.3"},
		{"npm:eslint@8.0.0", "eslint", "8.0.0"},
		{"npm:eslint", "eslint", ""},
		{"yarn:turbo@latest", "turbo", "latest"},
		{"npm:@scope/pkg@1.0.0", "@scope/pkg", "1.0.0"},
		{"pnpm:@scope/pkg", "@scope/pkg", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			name, ver := JSPMPackageName(tt.input)
			if name != tt.expectedName || ver != tt.expectedVer {
				t.Errorf("JSPMPackageName(%q) = (%q, %q), want (%q, %q)",
					tt.input, name, ver, tt.expectedName, tt.expectedVer)
			}
		})
	}
}
