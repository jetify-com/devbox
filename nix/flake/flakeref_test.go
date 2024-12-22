package flake

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFlakeRefString(t *testing.T) {
	cases := map[Ref]string{
		{}: "",

		// Path references.
		{Type: TypePath, Path: "."}:                "path:.",
		{Type: TypePath, Path: "./"}:               "path:.",
		{Type: TypePath, Path: "./flake"}:          "path:flake",
		{Type: TypePath, Path: "./relative/flake"}: "path:relative/flake",
		{Type: TypePath, Path: "/"}:                "path:/",
		{Type: TypePath, Path: "/flake"}:           "path:/flake",
		{Type: TypePath, Path: "/absolute/flake"}:  "path:/absolute/flake",

		// Path references with escapes.
		{Type: TypePath, Path: "%"}:                 "path:%25",
		{Type: TypePath, Path: "/%2F"}:              "path:/%252F",
		{Type: TypePath, Path: "./Ûñî©ôδ€/flake\n"}: "path:%C3%9B%C3%B1%C3%AE%C2%A9%C3%B4%CE%B4%E2%82%AC/flake%0A",
		{Type: TypePath, Path: "/Ûñî©ôδ€/flake\n"}:  "path:/%C3%9B%C3%B1%C3%AE%C2%A9%C3%B4%CE%B4%E2%82%AC/flake%0A",

		// Indirect references.
		{Type: TypeIndirect, ID: "indirect"}:                                                              "flake:indirect",
		{Type: TypeIndirect, ID: "indirect", Dir: "sub/dir"}:                                              "flake:indirect?dir=sub%2Fdir",
		{Type: TypeIndirect, ID: "indirect", Ref: "ref"}:                                                  "flake:indirect/ref",
		{Type: TypeIndirect, ID: "indirect", Ref: "my/ref"}:                                               "flake:indirect/my%2Fref",
		{Type: TypeIndirect, ID: "indirect", Rev: "5233fd2ba76a3accb5aaa999c00509a11fd0793c"}:             "flake:indirect/5233fd2ba76a3accb5aaa999c00509a11fd0793c",
		{Type: TypeIndirect, ID: "indirect", Ref: "ref", Rev: "5233fd2ba76a3accb5aaa999c00509a11fd0793c"}: "flake:indirect/ref/5233fd2ba76a3accb5aaa999c00509a11fd0793c",

		// GitHub references.
		{Type: TypeGitHub, Owner: "NixOS", Repo: "nix"}:                                                               "github:NixOS/nix",
		{Type: TypeGitHub, Owner: "NixOS", Repo: "nix", Ref: "v1.2.3"}:                                                "github:NixOS/nix/v1.2.3",
		{Type: TypeGitHub, Owner: "NixOS", Repo: "nix", Ref: "my/ref"}:                                                "github:NixOS/nix/my%2Fref",
		{Type: TypeGitHub, Owner: "NixOS", Repo: "nix", Ref: "5233fd2ba76a3accb5aaa999c00509a11fd0793c"}:              "github:NixOS/nix/5233fd2ba76a3accb5aaa999c00509a11fd0793c",
		{Type: TypeGitHub, Owner: "NixOS", Repo: "nix", Ref: "5233fd2bb76a3accb5aaa999c00509a11fd0793z"}:              "github:NixOS/nix/5233fd2bb76a3accb5aaa999c00509a11fd0793z",
		{Type: TypeGitHub, Owner: "NixOS", Repo: "nix", Rev: "5233fd2ba76a3accb5aaa999c00509a11fd0793c", Ref: "main"}: "github:NixOS/nix/5233fd2ba76a3accb5aaa999c00509a11fd0793c",
		{Type: TypeGitHub, Owner: "NixOS", Repo: "nix", Dir: "sub/dir"}:                                               "github:NixOS/nix?dir=sub%2Fdir",
		{Type: TypeGitHub, Owner: "NixOS", Repo: "nix", Dir: "sub/dir", Host: "example.com"}:                          "github:NixOS/nix?dir=sub%2Fdir&host=example.com",

		// Git references.
		{Type: TypeGit, Host: "example.com", Owner: "repo", Repo: "flake"}:                                                                                 "git://example.com/repo/flake",
		{Type: TypeHttps, Host: "example.com", Owner: "repo", Repo: "flake"}:                                                                               "git+https://example.com/repo/flake",
		{Type: TypeSSH, Host: "example.com", Owner: "repo", Repo: "flake"}:                                                                                 "git+ssh://git@example.com/repo/flake",
		{Type: TypeGit, Owner: "repo", Repo: "flake"}:                                                                                                      "git:/repo/flake",
		{Type: TypeFile, Owner: "repo", Repo: "flake"}:                                                                                                     "git+file:///repo/flake",
		{Type: TypeSSH, Host: "example.com", Owner: "repo", Repo: "flake", Ref: "my/ref", Rev: "e486d8d40e626a20e06d792db8cc5ac5aba9a5b4"}:                 "git+ssh://git@example.com/repo/flake?ref=my%2Fref&rev=e486d8d40e626a20e06d792db8cc5ac5aba9a5b4",
		{Type: TypeSSH, Host: "example.com", Owner: "repo", Repo: "flake", Ref: "my/ref", Rev: "e486d8d40e626a20e06d792db8cc5ac5aba9a5b4", Dir: "sub/dir"}: "git+ssh://git@example.com/repo/flake?dir=sub%2Fdir&ref=my%2Fref&rev=e486d8d40e626a20e06d792db8cc5ac5aba9a5b4",
		{Type: TypeGit, Owner: "repo", Repo: "flake", Ref: "my/ref", Rev: "e486d8d40e626a20e06d792db8cc5ac5aba9a5b4", Dir: "sub/dir"}:                      "", // "git:/repo/flake?dir=sub%2Fdir&ref=my%2Fref&rev=e486d8d40e626a20e06d792db8cc5ac5aba9a5b4", // how is this supposed to be a valid URL? There isn't a hostname in it...

		// Tarball references.
		{Type: TypeTarball, Host: "example.com", Owner: "flake", URL: "http://example.com/flake"}:                  "tarball+http://example.com/flake",
		{Type: TypeTarball, Host: "example.com", Owner: "flake", URL: "https://example.com/flake"}:                 "tarball+https://example.com/flake",
		{Type: TypeTarball, Host: "example.com", Owner: "flake", URL: "https://example.com/flake", Dir: "sub/dir"}: "tarball+https://example.com/flake?dir=sub%2Fdir",
		{Type: TypeTarball, URL: "file:///home/flake"}:                                                             "tarball+file:///home/flake",

		// File URL references.
		{Type: TypePath, URL: "file:///flake"}:                                              "file+file:///flake",
		{Type: TypePath, URL: "http://example.com/flake"}:                                   "file+http://example.com/flake",
		{Type: TypePath, URL: "http://example.com/flake.git"}:                               "file+http://example.com/flake.git",
		{Type: TypePath, URL: "http://example.com/flake.tar?dir=sub%2Fdir", Dir: "sub/dir"}: "file+http://example.com/flake.tar?dir=sub%2Fdir",
	}

	for ref, want := range cases {
		t.Run(want, func(t *testing.T) {
			t.Logf("input = %#v", ref)
			got := ref.String()
			if got != want {
				t.Errorf("got %#q, want %#q", got, want)
			}
		})
	}
}

func TestParseFlakeInstallable(t *testing.T) {
	cases := map[string]Installable{
		// Empty string is not a valid installable.
		"": {},

		// Not a path and not a valid URL.
		"://bad/url": {},

		".":             {Ref: Ref{Type: TypePath, Path: "."}},
		".#app":         {AttrPath: "app", Ref: Ref{Type: TypePath, Path: "."}},
		".#app^out":     {AttrPath: "app", Outputs: "out", Ref: Ref{Type: TypePath, Path: "."}},
		".#app^out,lib": {AttrPath: "app", Outputs: "lib,out", Ref: Ref{Type: TypePath, Path: "."}},
		".#app^*":       {AttrPath: "app", Outputs: "*", Ref: Ref{Type: TypePath, Path: "."}},
		".^*":           {Outputs: "*", Ref: Ref{Type: TypePath, Path: "."}},

		"./flake":             {Ref: Ref{Type: TypePath, Path: "./flake"}},
		"./flake#app":         {AttrPath: "app", Ref: Ref{Type: TypePath, Path: "./flake"}},
		"./flake#app^out":     {AttrPath: "app", Outputs: "out", Ref: Ref{Type: TypePath, Path: "./flake"}},
		"./flake#app^out,lib": {AttrPath: "app", Outputs: "lib,out", Ref: Ref{Type: TypePath, Path: "./flake"}},
		"./flake^out":         {Outputs: "out", Ref: Ref{Type: TypePath, Path: "./flake"}},

		"indirect":            {Ref: Ref{Type: TypeIndirect, ID: "indirect"}},
		"nixpkgs#app":         {AttrPath: "app", Ref: Ref{Type: TypeIndirect, ID: "nixpkgs"}},
		"nixpkgs#app^out":     {AttrPath: "app", Outputs: "out", Ref: Ref{Type: TypeIndirect, ID: "nixpkgs"}},
		"nixpkgs#app^out,lib": {AttrPath: "app", Outputs: "lib,out", Ref: Ref{Type: TypeIndirect, ID: "nixpkgs"}},
		"nixpkgs^out":         {Outputs: "out", Ref: Ref{Type: TypeIndirect, ID: "nixpkgs"}},

		"%23#app":       {AttrPath: "app", Ref: Ref{Type: TypeIndirect, ID: "#"}},
		"./%23#app":     {AttrPath: "app", Ref: Ref{Type: TypePath, Path: "./#"}},
		"/%23#app":      {AttrPath: "app", Ref: Ref{Type: TypePath, Path: "/#"}},
		"path:/%23#app": {AttrPath: "app", Ref: Ref{Type: TypePath, Path: "/#"}},

		"http://example.com/%23.tar#app":   {AttrPath: "app", Ref: Ref{Type: TypeTarball, URL: "http://example.com/%23.tar"}},
		"file:///flake#app":                {AttrPath: "app", Ref: Ref{Type: TypeFile, URL: "file:///flake"}},
		"git://example.com/repo/flake#app": {AttrPath: "app", Ref: Ref{Type: TypeGit, URL: "git://example.com/repo/flake"}},
	}

	for installable, want := range cases {
		t.Run(installable, func(t *testing.T) {
			got, err := ParseInstallable(installable)
			if diff := cmp.Diff(want, got); diff != "" {
				if err != nil {
					t.Errorf("got error: %s", err)
				}
				t.Errorf("wrong installable (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFlakeInstallableString(t *testing.T) {
	cases := map[Installable]string{
		{}: "",

		// No attribute or outputs.
		{Ref: Ref{Type: TypePath, Path: "."}}:          "path:.",
		{Ref: Ref{Type: TypePath, Path: "./flake"}}:    "path:flake",
		{Ref: Ref{Type: TypePath, Path: "/flake"}}:     "path:/flake",
		{Ref: Ref{Type: TypeIndirect, ID: "indirect"}}: "flake:indirect",

		// Attribute without outputs.
		{AttrPath: "app", Ref: Ref{Type: TypePath, Path: "."}}:            "path:.#app",
		{AttrPath: "my#app", Ref: Ref{Type: TypePath, Path: "."}}:         "path:.#my%23app",
		{AttrPath: "app", Ref: Ref{Type: TypePath, Path: "./flake"}}:      "path:flake#app",
		{AttrPath: "my#app", Ref: Ref{Type: TypePath, Path: "./flake"}}:   "path:flake#my%23app",
		{AttrPath: "app", Ref: Ref{Type: TypePath, Path: "/flake"}}:       "path:/flake#app",
		{AttrPath: "my#app", Ref: Ref{Type: TypePath, Path: "/flake"}}:    "path:/flake#my%23app",
		{AttrPath: "app", Ref: Ref{Type: TypeIndirect, ID: "nixpkgs"}}:    "flake:nixpkgs#app",
		{AttrPath: "my#app", Ref: Ref{Type: TypeIndirect, ID: "nixpkgs"}}: "flake:nixpkgs#my%23app",

		// Attribute with single output.
		{AttrPath: "app", Outputs: "out", Ref: Ref{Type: TypePath, Path: "."}}:         "path:.#app^out",
		{AttrPath: "app", Outputs: "out", Ref: Ref{Type: TypePath, Path: "./flake"}}:   "path:flake#app^out",
		{AttrPath: "app", Outputs: "out", Ref: Ref{Type: TypePath, Path: "/flake"}}:    "path:/flake#app^out",
		{AttrPath: "app", Outputs: "out", Ref: Ref{Type: TypeIndirect, ID: "nixpkgs"}}: "flake:nixpkgs#app^out",

		// Attribute with multiple outputs.
		{AttrPath: "app", Outputs: "out,lib", Ref: Ref{Type: TypePath, Path: "."}}:         "path:.#app^lib,out",
		{AttrPath: "app", Outputs: "out,lib", Ref: Ref{Type: TypePath, Path: "./flake"}}:   "path:flake#app^lib,out",
		{AttrPath: "app", Outputs: "out,lib", Ref: Ref{Type: TypePath, Path: "/flake"}}:    "path:/flake#app^lib,out",
		{AttrPath: "app", Outputs: "out,lib", Ref: Ref{Type: TypeIndirect, ID: "nixpkgs"}}: "flake:nixpkgs#app^lib,out",

		// Outputs are cleaned and sorted.
		{AttrPath: "app", Outputs: "out,lib", Ref: Ref{Type: TypePath, Path: "."}}:       "path:.#app^lib,out",
		{AttrPath: "app", Outputs: "lib,out", Ref: Ref{Type: TypePath, Path: "./flake"}}: "path:flake#app^lib,out",
		{AttrPath: "app", Outputs: "out,,", Ref: Ref{Type: TypePath, Path: "/flake"}}:    "path:/flake#app^out",
		{AttrPath: "app", Outputs: ",lib,out", Ref: Ref{Type: TypePath, Path: "/flake"}}: "path:/flake#app^lib,out",
		{AttrPath: "app", Outputs: ",", Ref: Ref{Type: TypeIndirect, ID: "nixpkgs"}}:     "flake:nixpkgs#app",

		// Wildcard replaces other outputs.
		{AttrPath: "app", Outputs: "*", Ref: Ref{Type: TypeIndirect, ID: "nixpkgs"}}:     "flake:nixpkgs#app^*",
		{AttrPath: "app", Outputs: "out,*", Ref: Ref{Type: TypeIndirect, ID: "nixpkgs"}}: "flake:nixpkgs#app^*",
		{AttrPath: "app", Outputs: ",*", Ref: Ref{Type: TypeIndirect, ID: "nixpkgs"}}:    "flake:nixpkgs#app^*",

		// Outputs are not percent-encoded.
		{AttrPath: "app", Outputs: "%", Ref: Ref{Type: TypeIndirect, ID: "nixpkgs"}}:   "flake:nixpkgs#app^%",
		{AttrPath: "app", Outputs: "/", Ref: Ref{Type: TypeIndirect, ID: "nixpkgs"}}:   "flake:nixpkgs#app^/",
		{AttrPath: "app", Outputs: "%2F", Ref: Ref{Type: TypeIndirect, ID: "nixpkgs"}}: "flake:nixpkgs#app^%2F",

		// Missing or invalid fields.
		{AttrPath: "app", Ref: Ref{Type: TypePath, URL: ""}}:     "",
		{AttrPath: "app", Ref: Ref{Type: TypeGit, URL: ""}}:      "",
		{AttrPath: "app", Ref: Ref{Type: TypeGitHub, Owner: ""}}: "",
		{AttrPath: "app", Ref: Ref{Type: TypeIndirect, ID: ""}}:  "",
		{AttrPath: "app", Ref: Ref{Type: TypePath, Path: ""}}:    "",
		{AttrPath: "app", Ref: Ref{Type: TypeTarball, URL: ""}}:  "",
	}

	for installable, want := range cases {
		t.Run(want, func(t *testing.T) {
			t.Logf("input = %#v", installable)
			got := installable.String()
			if got != want {
				t.Errorf("got %#q, want %#q", got, want)
			}
		})
	}
}

func TestBuildQueryString(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("wanted panic for odd-number of key-value parameters")
		}
	}()

	// staticcheck impressively catches buildQueryString calls that have an
	// odd number of parameters. Build the slice in a convoluted way to
	// throw it off and suppress the warning (gopls doesn't have nolint
	// directives).
	var elems []string
	elems = append(elems, "1")
	appendQueryString(nil, elems...)
}
