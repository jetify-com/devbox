package nix

import (
	"testing"
)

func TestIsUnfreeAllowed(t *testing.T) {
	cases := map[string]bool{
		"":      false,
		"0":     false,
		"false": false,
		"1":     true,
		"true":  true,
	}
	for value, want := range cases {
		t.Run(value, func(t *testing.T) {
			t.Setenv("NIXPKGS_ALLOW_UNFREE", value)
			if got := IsUnfreeAllowed(); got != want {
				t.Errorf("IsUnfreeAllowed() with NIXPKGS_ALLOW_UNFREE=%q = %v, want %v", value, got, want)
			}
		})
	}
}

// TestUsePrintDevEnvImpure verifies the fix for
// https://github.com/jetify-com/devbox/issues/2196: print-dev-env must run
// with --impure whenever the user opts into unfree or insecure packages so
// those environment variables are honored.
func TestUsePrintDevEnvImpure(t *testing.T) {
	cases := []struct {
		name          string
		allowUnfree   bool
		allowInsecure bool
		want          bool
	}{
		{name: "neither opted in", want: false},
		{name: "unfree opted in", allowUnfree: true, want: true},
		{name: "insecure opted in", allowInsecure: true, want: true},
		{name: "both opted in", allowUnfree: true, allowInsecure: true, want: true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			// Keep the result independent of any ambient feature-flag env.
			t.Setenv("DEVBOX_FEATURE_IMPURE_PRINT_DEV_ENV", "0")
			if got := usePrintDevEnvImpure(tt.allowUnfree, tt.allowInsecure); got != tt.want {
				t.Errorf("usePrintDevEnvImpure(%v, %v) = %v, want %v",
					tt.allowUnfree, tt.allowInsecure, got, tt.want)
			}
		})
	}
}

// TestUsePrintDevEnvImpureFeatureFlag verifies that the IMPURE_PRINT_DEV_ENV
// feature flag forces impure mode even when the user hasn't opted into unfree
// or insecure packages.
func TestUsePrintDevEnvImpureFeatureFlag(t *testing.T) {
	t.Setenv("DEVBOX_FEATURE_IMPURE_PRINT_DEV_ENV", "1")
	if !usePrintDevEnvImpure(false, false) {
		t.Error("usePrintDevEnvImpure(false, false) = false with feature flag set, want true")
	}
}

func TestParseInsecurePackagesFromExitError(t *testing.T) {
	errorText := `
  at /nix/store/xwl0am98klc8mz074jdyvpnyc6vwzlla-source/lib/customisation.nix:267:17:

          266|     in commonAttrs // {
          267|       drvPath = assert condition; drv.drvPath;
             |                 ^
          268|       outPath = assert condition; drv.outPath;

       … while evaluating the attribute 'handled'

         at /nix/store/xwl0am98klc8mz074jdyvpnyc6vwzlla-source/pkgs/stdenv/generic/check-meta.nix:490:7:

          489|       # or, alternatively, just output a warning message.
          490|       handled =
             |       ^
          491|         (

       (stack trace truncated; use '--show-trace' to show the full trace)

       error: Package ‘python-2.7.18.7’ in /nix/store/xwl0am98klc8mz074jdyvpnyc6vwzlla-source/pkgs/development/interpreters/python/cpython/2.7/default.nix:335 is marked as insecure, refusing to evaluate.


       Known issues:
        - Python 2.7 has reached its end of life after 2020-01-01. See https://www.python.org/doc/sunset-python-2/.

       You can install it anyway by allowing this package, using the
       following methods:

       a) To temporarily allow all insecure packages, you can use an environment
          variable for a single invocation of the nix tools:

            $ export NIXPKGS_ALLOW_INSECURE=1

          Note: When using nix shell, nix build, nix develop, etc with a flake,
                then pass --impure in order to allow use of environment variables.

       b) for nixos-rebuild you can add ‘python-2.7.18.7’ to
          nixpkgs.config.permittedInsecurePackages in the configuration.nix,
          like so:

            {
              nixpkgs.config.permittedInsecurePackages = [
                "python-2.7.18.7"
              ];
            }

       c) For nix-env, nix-build, nix-shell or any other Nix command you can add
          ‘python-2.7.18.7’ to permittedInsecurePackages in
          ~/.config/nixpkgs/config.nix, like so:

            {
              permittedInsecurePackages = [
                "python-2.7.18.7"
              ];
              `
	packages := parseInsecurePackagesFromExitError(errorText)
	if len(packages) != 1 {
		t.Errorf("Expected 1 package, got %d", len(packages))
	}
	if packages[0] != "python-2.7.18.7" {
		t.Errorf("Expected package 'python-2.7.18.7', got %s", packages[0])
	}
}
