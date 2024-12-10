//nolint:dupl
package nix

import (
	"slices"
	"testing"
)

func TestParseVersionInfo(t *testing.T) {
	raw := `nix (Nix) 2.21.2
System type: aarch64-darwin
Additional system types: x86_64-darwin
Features: gc, signed-caches
System configuration file: /etc/nix/nix.conf
User configuration files: /Users/nobody/.config/nix/nix.conf:/etc/xdg/nix/nix.conf
Store directory: /nix/store
State directory: /nix/var/nix
Data directory: /nix/store/m0ns07v8by0458yp6k30rfq1rs3kaz6g-nix-2.21.2/share
`

	info, err := parseInfo([]byte(raw))
	if err != nil {
		t.Error("got parse error:", err)
	}
	if got, want := info.Name, "nix"; got != want {
		t.Errorf("got Name = %q, want %q", got, want)
	}
	if got, want := info.Version, "2.21.2"; got != want {
		t.Errorf("got Version = %q, want %q", got, want)
	}
	if got, want := info.System, "aarch64-darwin"; got != want {
		t.Errorf("got System = %q, want %q", got, want)
	}
	if got, want := info.ExtraSystems, []string{"x86_64-darwin"}; !slices.Equal(got, want) {
		t.Errorf("got ExtraSystems = %q, want %q", got, want)
	}
	if got, want := info.Features, []string{"gc", "signed-caches"}; !slices.Equal(got, want) {
		t.Errorf("got Features = %q, want %q", got, want)
	}
	if got, want := info.SystemConfig, "/etc/nix/nix.conf"; got != want {
		t.Errorf("got SystemConfig = %q, want %q", got, want)
	}
	if got, want := info.UserConfigs, []string{"/Users/nobody/.config/nix/nix.conf", "/etc/xdg/nix/nix.conf"}; !slices.Equal(got, want) {
		t.Errorf("got UserConfigs = %q, want %q", got, want)
	}
	if got, want := info.StoreDir, "/nix/store"; got != want {
		t.Errorf("got StoreDir = %q, want %q", got, want)
	}
	if got, want := info.StateDir, "/nix/var/nix"; got != want {
		t.Errorf("got StateDir = %q, want %q", got, want)
	}
	if got, want := info.DataDir, "/nix/store/m0ns07v8by0458yp6k30rfq1rs3kaz6g-nix-2.21.2/share"; got != want {
		t.Errorf("got DataDir = %q, want %q", got, want)
	}
}

func TestParseLixVersionInfo(t *testing.T) {
	raw := `nix (Lix, like Nix) 2.90.0-beta.1
System type: aarch64-darwin
Additional system types: x86_64-darwin
Features: gc, signed-caches
System configuration file: /etc/nix/nix.conf
User configuration files: /Users/nobody/.config/nix/nix.conf:/etc/xdg/nix/nix.conf
Store directory: /nix/store
State directory: /nix/var/nix
Data directory: /nix/store/12asl5a17ffj78njcy2fj31v59rdmanx-lix-2.90-beta.1/share
`

	info, err := parseInfo([]byte(raw))
	if err != nil {
		t.Error("got parse error:", err)
	}
	if got, want := info.Name, "nix"; got != want {
		t.Errorf("got Name = %q, want %q", got, want)
	}
	if got, want := info.Version, "2.90.0-beta.1"; got != want {
		t.Errorf("got Version = %q, want %q", got, want)
	}
	if got, want := info.System, "aarch64-darwin"; got != want {
		t.Errorf("got System = %q, want %q", got, want)
	}
	if got, want := info.ExtraSystems, []string{"x86_64-darwin"}; !slices.Equal(got, want) {
		t.Errorf("got ExtraSystems = %q, want %q", got, want)
	}
	if got, want := info.Features, []string{"gc", "signed-caches"}; !slices.Equal(got, want) {
		t.Errorf("got Features = %q, want %q", got, want)
	}
	if got, want := info.SystemConfig, "/etc/nix/nix.conf"; got != want {
		t.Errorf("got SystemConfig = %q, want %q", got, want)
	}
	if got, want := info.UserConfigs, []string{"/Users/nobody/.config/nix/nix.conf", "/etc/xdg/nix/nix.conf"}; !slices.Equal(got, want) {
		t.Errorf("got UserConfigs = %q, want %q", got, want)
	}
	if got, want := info.StoreDir, "/nix/store"; got != want {
		t.Errorf("got StoreDir = %q, want %q", got, want)
	}
	if got, want := info.StateDir, "/nix/var/nix"; got != want {
		t.Errorf("got StateDir = %q, want %q", got, want)
	}
	if got, want := info.DataDir, "/nix/store/12asl5a17ffj78njcy2fj31v59rdmanx-lix-2.90-beta.1/share"; got != want {
		t.Errorf("got DataDir = %q, want %q", got, want)
	}
}

func TestParseVersionInfoShort(t *testing.T) {
	cases := []struct {
		in      string
		name    string
		version string
	}{
		{"nix (Nix) 2.21.2", "nix", "2.21.2"},
		{"nix (Nix) 2.23.0pre20240526_7de033d6", "nix", "2.23.0pre20240526_7de033d6"},
		{"command (Nix) name (Nix) 2.21.2", "command (Nix) name", "2.21.2"},
		{"nix (Lix, like Nix) 2.90.0-beta.1", "nix", "2.90.0-beta.1"},
	}

	for _, tt := range cases {
		t.Run(tt.in, func(t *testing.T) {
			got, err := parseInfo([]byte(tt.in))
			if err != nil {
				t.Error("got parse error:", err)
			}
			if got.Name != tt.name {
				t.Errorf("got Name = %q, want %q", got.Name, tt.name)
			}
			if got.Version != tt.version {
				t.Errorf("got Version = %q, want %q", got.Version, tt.version)
			}
		})
	}
}

func TestParseVersionInfoError(t *testing.T) {
	t.Run("NilOutput", func(t *testing.T) {
		_, err := parseInfo(nil)
		if err == nil {
			t.Error("want non-nil error")
		}
	})
	t.Run("EmptyOutput", func(t *testing.T) {
		_, err := parseInfo([]byte{})
		if err == nil {
			t.Error("want non-nil error")
		}
	})
	t.Run("MissingVersionOutput", func(t *testing.T) {
		_, err := parseInfo([]byte("nix output without a version"))
		if err == nil {
			t.Error("want non-nil error")
		}
	})
}

func TestVersionInfoAtLeast(t *testing.T) {
	info := Info{}
	if info.AtLeast(Version2_12) {
		t.Errorf("got empty current version >= %s", Version2_12)
	}

	info.Version = Version2_13
	if !info.AtLeast(Version2_12) {
		t.Errorf("got %s < %s", info.Version, Version2_12)
	}
	if !info.AtLeast(Version2_13) {
		t.Errorf("got %s < %s", info.Version, Version2_13)
	}
	if info.AtLeast(Version2_14) {
		t.Errorf("got %s >= %s", info.Version, Version2_14)
	}

	// https://github.com/jetify-com/devbox/issues/2128
	info.Version = "2.23.0pre20240526_7de033d6"
	if !info.AtLeast(Version2_12) {
		t.Errorf("got %s < %s", info.Version, Version2_12)
	}
	if info.AtLeast("2.23.0") {
		t.Errorf("got %s > %s", info.Version, "2.23.0")
	}
	if info.AtLeast("2.24.0") {
		t.Errorf("got %s > %s", info.Version, "2.24.0")
	}
	if info.AtLeast("2.23.0-pre.99999999") {
		t.Errorf("got %s > %s", info.Version, "2.23.0-pre.99999999")
	}
	if !info.AtLeast("2.23.0-pre.1") {
		t.Errorf("got %s < %s", info.Version, "2.23.0-pre.1")
	}

	t.Run("ArgEmptyPanic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("want panic for empty version")
			}
		}()
		info.AtLeast("")
	})
	t.Run("ArgInvalidPanic", func(t *testing.T) {
		v := "notasemver"
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("want panic for invalid version %q", v)
			}
		}()
		info.AtLeast(v)
	})
}
