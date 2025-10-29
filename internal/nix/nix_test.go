package nix

import (
	"testing"
)

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
