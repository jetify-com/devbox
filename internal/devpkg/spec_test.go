package devpkg

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.jetpack.io/devbox/nix/flake"
	"go.jetpack.io/pkg/runx/impl/types"
)

func TestParsePackageSpec(t *testing.T) {
	cases := []struct {
		in   string
		want PackageSpec
	}{
		{in: "", want: PackageSpec{}},
		{in: "mail:nixpkgs#go", want: PackageSpec{}},

		// Common name@version strings.
		{
			in: "go", want: PackageSpec{
				Name: "go", Version: "latest",
				Installable:         mustFlake(t, "flake:go"),
				AttrPathInstallable: mustFlake(t, "nixpkgs#go"),
			},
		},
		{
			in: "go@latest", want: PackageSpec{
				Name: "go", Version: "latest",
				Installable:         mustFlake(t, "flake:go@latest"),
				AttrPathInstallable: mustFlake(t, "nixpkgs#go@latest"),
			},
		},
		{
			in: "go@1.22.0", want: PackageSpec{
				Name: "go", Version: "1.22.0",
				Installable:         mustFlake(t, "flake:go@1.22.0"),
				AttrPathInstallable: mustFlake(t, "nixpkgs#go@1.22.0"),
			},
		},

		// name@version splitting edge-cases.
		{
			in: "emacsPackages.@@latest", want: PackageSpec{
				Name: "emacsPackages.@", Version: "latest",
				Installable:         mustFlake(t, "flake:emacsPackages.@@latest"),
				AttrPathInstallable: mustFlake(t, "nixpkgs#emacsPackages.@@latest"),
			},
		},
		{
			in: "emacsPackages.@", want: PackageSpec{
				Name: "emacsPackages.@", Version: "latest",
				Installable:         mustFlake(t, "flake:emacsPackages.@"),
				AttrPathInstallable: mustFlake(t, "nixpkgs#emacsPackages.@"),
			},
		},
		{
			in: "@angular/cli", want: PackageSpec{
				Name: "@angular/cli", Version: "latest",
				Installable:         mustFlake(t, "flake:@angular/cli"),
				AttrPathInstallable: mustFlake(t, "nixpkgs#@angular/cli"),
			},
		},
		{
			in: "nodePackages.@angular/cli", want: PackageSpec{
				Name: "nodePackages.", Version: "angular/cli",
				Installable:         mustFlake(t, "flake:nodePackages.@angular/cli"),
				AttrPathInstallable: mustFlake(t, "nixpkgs#nodePackages.@angular/cli"),
			},
		},

		// Flake installables.
		{
			in:   "nixpkgs#go",
			want: PackageSpec{Installable: mustFlake(t, "flake:nixpkgs#go")},
		},
		{
			in:   "flake:nixpkgs",
			want: PackageSpec{Installable: mustFlake(t, "flake:nixpkgs")},
		},
		{
			in:   "flake:nixpkgs#go",
			want: PackageSpec{Installable: mustFlake(t, "flake:nixpkgs#go")},
		},
		{
			in:   "./my-php-flake",
			want: PackageSpec{Installable: mustFlake(t, "path:./my-php-flake")},
		},
		{
			in:   "./my-php-flake#hello",
			want: PackageSpec{Installable: mustFlake(t, "path:./my-php-flake#hello")},
		},
		{
			in:   "/my-php-flake",
			want: PackageSpec{Installable: mustFlake(t, "path:/my-php-flake")},
		},
		{
			in:   "/my-php-flake#hello",
			want: PackageSpec{Installable: mustFlake(t, "path:/my-php-flake#hello")},
		},
		{
			in:   "path:my-php-flake",
			want: PackageSpec{Installable: mustFlake(t, "path:my-php-flake")},
		},
		{
			in:   "path:my-php-flake#hello",
			want: PackageSpec{Installable: mustFlake(t, "path:my-php-flake#hello")},
		},
		{
			in:   "github:F1bonacc1/process-compose/v0.43.1",
			want: PackageSpec{Installable: mustFlake(t, "github:F1bonacc1/process-compose/v0.43.1")},
		},
		{
			in:   "github:nixos/nixpkgs/5233fd2ba76a3accb5aaa999c00509a11fd0793c#hello",
			want: PackageSpec{Installable: mustFlake(t, "github:nixos/nixpkgs/5233fd2ba76a3accb5aaa999c00509a11fd0793c#hello")},
		},
		{
			in: "mail:nixpkgs",
			want: PackageSpec{
				Name: "mail:nixpkgs", Version: "latest",
				AttrPathInstallable: mustFlake(t, "nixpkgs#mail:nixpkgs"),
			},
		},

		// RunX
		{
			in: "runx:golangci/golangci-lint", want: PackageSpec{
				RunX: types.PkgRef{
					Owner:   "golangci",
					Repo:    "golangci-lint",
					Version: "latest",
				},
			},
		},
		{
			in: "runx:golangci/golangci-lint@1.2.3", want: PackageSpec{
				RunX: types.PkgRef{
					Owner:   "golangci",
					Repo:    "golangci-lint",
					Version: "1.2.3",
				},
			},
		},

		// RunX missing scheme.
		{
			in: "golangci/golangci-lint", want: PackageSpec{
				Name: "golangci/golangci-lint", Version: "latest",
				Installable:         mustFlake(t, "flake:golangci/golangci-lint"),
				AttrPathInstallable: mustFlake(t, "nixpkgs#golangci/golangci-lint"),
			},
		},
		{
			in: "golangci/golangci-lint@1.2.3", want: PackageSpec{
				Name: "golangci/golangci-lint", Version: "1.2.3",
				Installable:         mustFlake(t, "flake:golangci/golangci-lint@1.2.3"),
				AttrPathInstallable: mustFlake(t, "nixpkgs#golangci/golangci-lint@1.2.3"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("in=%s", tc.in), func(t *testing.T) {
			got := ParsePackageSpec(tc.in, "")
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("wrong PackageSpec for %q (-want +got):\n%s", tc.in, diff)
			}
		})
	}
}

// TestParseDeprecatedPackageSpec tests parsing behavior when the deprecated
// nixpkgs.commit field is set to nixpkgs-unstable. It's split into a separate
// test in case we ever drop support for nixpkgs.commit entirely.
func TestParseDeprecatedPackageSpec(t *testing.T) {
	nixpkgsCommit := flake.Ref{Type: flake.TypeIndirect, ID: "nixpkgs", Ref: "nixpkgs-unstable"}
	cases := []struct {
		in   string
		want PackageSpec
	}{
		{in: "", want: PackageSpec{}},

		// Parses Devbox package when @version specified.
		{
			in: "go@latest", want: PackageSpec{
				Name: "go", Version: "latest",
				AttrPathInstallable: mustFlake(t, "nixpkgs/nixpkgs-unstable#go@latest"),
			},
		},
		{
			in: "go@1.22.0", want: PackageSpec{
				Name: "go", Version: "1.22.0",
				AttrPathInstallable: mustFlake(t, "nixpkgs/nixpkgs-unstable#go@1.22.0"),
			},
		},

		// Missing @version does not imply @latest and is not a flake reference.
		{in: "go", want: PackageSpec{AttrPathInstallable: mustFlake(t, "nixpkgs/nixpkgs-unstable#go")}},
		{in: "cachix", want: PackageSpec{AttrPathInstallable: mustFlake(t, "nixpkgs/nixpkgs-unstable#cachix")}},

		// Unambiguous flake reference should not be parsed as an attribute path.
		{in: "flake:cachix", want: PackageSpec{Installable: mustFlake(t, "flake:cachix#")}},
		{in: "./flake", want: PackageSpec{Installable: mustFlake(t, "path:./flake")}},
		{in: "path:flake", want: PackageSpec{Installable: mustFlake(t, "path:flake")}},
		{in: "nixpkgs#go", want: PackageSpec{Installable: mustFlake(t, "nixpkgs#go")}},
		{in: "nixpkgs/branch#go", want: PackageSpec{Installable: mustFlake(t, "nixpkgs/branch#go")}},

		// // RunX unaffected by nixpkgs.commit.
		{in: "runx:golangci/golangci-lint", want: PackageSpec{RunX: types.PkgRef{Owner: "golangci", Repo: "golangci-lint", Version: "latest"}}},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("in=%s", tc.in), func(t *testing.T) {
			got := ParsePackageSpec(tc.in, nixpkgsCommit.Ref)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("wrong PackageSpec for %q (-want +got):\n%s", tc.in, diff)
			}
		})
	}
}

// mustFlake parses s as a [flake.Installable] and fails the test if there's an
// error. It allows using the string form of a flake in test cases so they're
// easier to read.
func mustFlake(t *testing.T, s string) flake.Installable {
	t.Helper()
	i, err := flake.ParseInstallable(s)
	if err != nil {
		t.Fatal("error parsing wanted flake installable:", err)
	}
	return i
}
