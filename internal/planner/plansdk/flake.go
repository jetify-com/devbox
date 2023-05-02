package plansdk

import (
	"strings"

	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/nix"
)

type FlakeInput struct {
	Name     string
	Packages []string
	URL      string
}

// IsNixpkgs returns true if the input is a nixpkgs flake of the form:
// github:NixOS/nixpkgs/...
//
// While there are many ways to specify this input, devbox always uses
// github:NixOS/nixpkgs/<hash> as the URL. If the user wishes to reference nixpkgs
// themselves, this function may not return true.
func (f *FlakeInput) IsNixpkgs() bool {
	return nix.IsGithubNixpkgsURL(f.URL)
}

func (f *FlakeInput) URLWithCaching() string {
	if !f.IsNixpkgs() {
		return f.URL
	}
	hash := nix.HashFromNixPkgsURL(f.URL)
	return GetNixpkgsInfo(hash).URL
}

func (f *FlakeInput) PkgImportName() string {
	return f.Name + "-pkgs"
}

func (f *FlakeInput) BuildInputs() []string {
	if !f.IsNixpkgs() {
		return lo.Map(f.Packages, func(pkg string, _ int) string {
			return f.Name + "." + pkg
		})
	}
	return lo.Map(f.Packages, func(pkg string, _ int) string {
		parts := strings.Split(pkg, ".")
		// Ugh, not sure if this is reliable?
		return f.PkgImportName() + "." + strings.Join(parts[2:], ".")
	})
}
