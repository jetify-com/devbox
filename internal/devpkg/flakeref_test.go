package devpkg

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestParseFlakeRef(t *testing.T) {
	// Test cases use the zero-value to check for invalid flakerefs because
	// we don't care about the specific error message.
	cases := map[string]FlakeRef{
		// Empty string is not a valid flake reference.
		"": {},

		// Not a path and not a valid URL.
		"://bad/url": {},

		// Invalid escape.
		"path:./relative/my%flake": {},

		// Path-like references start with a '.' or '/'.
		// This distinguishes them from indirect references
		// (./nixpkgs is a directory; nixpkgs is an indirect).
		".":                {Type: "path", Path: "."},
		"./":               {Type: "path", Path: "./"},
		"./flake":          {Type: "path", Path: "./flake"},
		"./relative/flake": {Type: "path", Path: "./relative/flake"},
		"/":                {Type: "path", Path: "/"},
		"/flake":           {Type: "path", Path: "/flake"},
		"/absolute/flake":  {Type: "path", Path: "/absolute/flake"},

		// Path-like references can have raw unicode characters unlike
		// path: URL references.
		"./Ûñî©ôδ€/flake\n": {Type: "path", Path: "./Ûñî©ôδ€/flake\n"},
		"/Ûñî©ôδ€/flake\n":  {Type: "path", Path: "/Ûñî©ôδ€/flake\n"},

		// Path-like references don't allow paths with a '?' or '#'.
		"./invalid#path": {},
		"./invalid?path": {},
		"/invalid#path":  {},
		"/invalid?path":  {},
		"/#":             {},
		"/?":             {},

		// URL-like path references.
		"path:":                      {Type: "path", Path: ""},
		"path:.":                     {Type: "path", Path: "."},
		"path:./":                    {Type: "path", Path: "./"},
		"path:./flake":               {Type: "path", Path: "./flake"},
		"path:./relative/flake":      {Type: "path", Path: "./relative/flake"},
		"path:./relative/my%20flake": {Type: "path", Path: "./relative/my flake"},
		"path:/":                     {Type: "path", Path: "/"},
		"path:/flake":                {Type: "path", Path: "/flake"},
		"path:/absolute/flake":       {Type: "path", Path: "/absolute/flake"},

		// URL-like paths can omit the "./" prefix for relative
		// directories.
		"path:flake":          {Type: "path", Path: "flake"},
		"path:relative/flake": {Type: "path", Path: "relative/flake"},

		// Indirect references.
		"flake:indirect":          {Type: "indirect", ID: "indirect"},
		"flake:indirect/ref":      {Type: "indirect", ID: "indirect", Ref: "ref"},
		"flake:indirect/my%2Fref": {Type: "indirect", ID: "indirect", Ref: "my/ref"},
		"flake:indirect/5233fd2ba76a3accb5aaa999c00509a11fd0793c":     {Type: "indirect", ID: "indirect", Rev: "5233fd2ba76a3accb5aaa999c00509a11fd0793c"},
		"flake:indirect/ref/5233fd2ba76a3accb5aaa999c00509a11fd0793c": {Type: "indirect", ID: "indirect", Ref: "ref", Rev: "5233fd2ba76a3accb5aaa999c00509a11fd0793c"},

		// Indirect references can omit their "indirect:" type prefix.
		"indirect":     {Type: "indirect", ID: "indirect"},
		"indirect/ref": {Type: "indirect", ID: "indirect", Ref: "ref"},
		"indirect/5233fd2ba76a3accb5aaa999c00509a11fd0793c":     {Type: "indirect", ID: "indirect", Rev: "5233fd2ba76a3accb5aaa999c00509a11fd0793c"},
		"indirect/ref/5233fd2ba76a3accb5aaa999c00509a11fd0793c": {Type: "indirect", ID: "indirect", Ref: "ref", Rev: "5233fd2ba76a3accb5aaa999c00509a11fd0793c"},

		// GitHub references.
		"github:NixOS/nix":            {Type: "github", Owner: "NixOS", Repo: "nix"},
		"github:NixOS/nix/v1.2.3":     {Type: "github", Owner: "NixOS", Repo: "nix", Ref: "v1.2.3"},
		"github:NixOS/nix?ref=v1.2.3": {Type: "github", Owner: "NixOS", Repo: "nix", Ref: "v1.2.3"},
		"github:NixOS/nix?ref=5233fd2ba76a3accb5aaa999c00509a11fd0793c": {Type: "github", Owner: "NixOS", Repo: "nix", Ref: "5233fd2ba76a3accb5aaa999c00509a11fd0793c"},
		"github:NixOS/nix/5233fd2ba76a3accb5aaa999c00509a11fd0793c":     {Type: "github", Owner: "NixOS", Repo: "nix", Rev: "5233fd2ba76a3accb5aaa999c00509a11fd0793c"},
		"github:NixOS/nix/5233fd2bb76a3accb5aaa999c00509a11fd0793z":     {Type: "github", Owner: "NixOS", Repo: "nix", Ref: "5233fd2bb76a3accb5aaa999c00509a11fd0793z"},
		"github:NixOS/nix?rev=5233fd2ba76a3accb5aaa999c00509a11fd0793c": {Type: "github", Owner: "NixOS", Repo: "nix", Rev: "5233fd2ba76a3accb5aaa999c00509a11fd0793c"},
		"github:NixOS/nix?host=example.com":                             {Type: "github", Owner: "NixOS", Repo: "nix", Host: "example.com"},

		// GitHub references with invalid ref + rev combinations.
		"github:NixOS/nix?ref=v1.2.3&rev=5233fd2ba76a3accb5aaa999c00509a11fd0793c":                               {},
		"github:NixOS/nix/v1.2.3?ref=v4.5.6":                                                                     {},
		"github:NixOS/nix/5233fd2ba76a3accb5aaa999c00509a11fd0793c?rev=e486d8d40e626a20e06d792db8cc5ac5aba9a5b4": {},
		"github:NixOS/nix/5233fd2ba76a3accb5aaa999c00509a11fd0793c?ref=v1.2.3":                                   {},

		// The github type allows clone-style URLs. The username and
		// host are ignored.
		"github://git@github.com/NixOS/nix":                                              {Type: "github", Owner: "NixOS", Repo: "nix"},
		"github://git@github.com/NixOS/nix/v1.2.3":                                       {Type: "github", Owner: "NixOS", Repo: "nix", Ref: "v1.2.3"},
		"github://git@github.com/NixOS/nix?ref=v1.2.3":                                   {Type: "github", Owner: "NixOS", Repo: "nix", Ref: "v1.2.3"},
		"github://git@github.com/NixOS/nix?ref=5233fd2ba76a3accb5aaa999c00509a11fd0793c": {Type: "github", Owner: "NixOS", Repo: "nix", Ref: "5233fd2ba76a3accb5aaa999c00509a11fd0793c"},
		"github://git@github.com/NixOS/nix?rev=5233fd2ba76a3accb5aaa999c00509a11fd0793c": {Type: "github", Owner: "NixOS", Repo: "nix", Rev: "5233fd2ba76a3accb5aaa999c00509a11fd0793c"},
		"github://git@github.com/NixOS/nix?host=example.com":                             {Type: "github", Owner: "NixOS", Repo: "nix", Host: "example.com"},

		// Git references.
		"git://example.com/repo/flake":         {Type: "git", URL: "git://example.com/repo/flake"},
		"git+https://example.com/repo/flake":   {Type: "git", URL: "https://example.com/repo/flake"},
		"git+ssh://git@example.com/repo/flake": {Type: "git", URL: "ssh://git@example.com/repo/flake"},
		"git:/repo/flake":                      {Type: "git", URL: "git:/repo/flake"},
		"git+file:///repo/flake":               {Type: "git", URL: "file:///repo/flake"},
		"git://example.com/repo/flake?ref=unstable&rev=e486d8d40e626a20e06d792db8cc5ac5aba9a5b4&dir=subdir": {Type: "git", URL: "git://example.com/repo/flake?dir=subdir", Ref: "unstable", Rev: "e486d8d40e626a20e06d792db8cc5ac5aba9a5b4", Dir: "subdir"},

		// Tarball references.
		"tarball+http://example.com/flake":  {Type: "tarball", URL: "http://example.com/flake"},
		"tarball+https://example.com/flake": {Type: "tarball", URL: "https://example.com/flake"},
		"tarball+file:///home/flake":        {Type: "tarball", URL: "file:///home/flake"},

		// Regular URLs have the tarball type if they have a known
		// archive extension:
		// .zip, .tar, .tgz, .tar.gz, .tar.xz, .tar.bz2 or .tar.zst
		"http://example.com/flake.zip":            {Type: "tarball", URL: "http://example.com/flake.zip"},
		"http://example.com/flake.tar":            {Type: "tarball", URL: "http://example.com/flake.tar"},
		"http://example.com/flake.tgz":            {Type: "tarball", URL: "http://example.com/flake.tgz"},
		"http://example.com/flake.tar.gz":         {Type: "tarball", URL: "http://example.com/flake.tar.gz"},
		"http://example.com/flake.tar.xz":         {Type: "tarball", URL: "http://example.com/flake.tar.xz"},
		"http://example.com/flake.tar.bz2":        {Type: "tarball", URL: "http://example.com/flake.tar.bz2"},
		"http://example.com/flake.tar.zst":        {Type: "tarball", URL: "http://example.com/flake.tar.zst"},
		"http://example.com/flake.tar?dir=subdir": {Type: "tarball", URL: "http://example.com/flake.tar?dir=subdir", Dir: "subdir"},
		"file:///flake.zip":                       {Type: "tarball", URL: "file:///flake.zip"},
		"file:///flake.tar":                       {Type: "tarball", URL: "file:///flake.tar"},
		"file:///flake.tgz":                       {Type: "tarball", URL: "file:///flake.tgz"},
		"file:///flake.tar.gz":                    {Type: "tarball", URL: "file:///flake.tar.gz"},
		"file:///flake.tar.xz":                    {Type: "tarball", URL: "file:///flake.tar.xz"},
		"file:///flake.tar.bz2":                   {Type: "tarball", URL: "file:///flake.tar.bz2"},
		"file:///flake.tar.zst":                   {Type: "tarball", URL: "file:///flake.tar.zst"},
		"file:///flake.tar?dir=subdir":            {Type: "tarball", URL: "file:///flake.tar?dir=subdir", Dir: "subdir"},

		// File URL references.
		"file+file:///flake":                           {Type: "file", URL: "file:///flake"},
		"file+http://example.com/flake":                {Type: "file", URL: "http://example.com/flake"},
		"file+http://example.com/flake.git":            {Type: "file", URL: "http://example.com/flake.git"},
		"file+http://example.com/flake.tar?dir=subdir": {Type: "file", URL: "http://example.com/flake.tar?dir=subdir", Dir: "subdir"},

		// Regular URLs have the file type if they don't have a known
		// archive extension.
		"http://example.com/flake":            {Type: "file", URL: "http://example.com/flake"},
		"http://example.com/flake.git":        {Type: "file", URL: "http://example.com/flake.git"},
		"http://example.com/flake?dir=subdir": {Type: "file", URL: "http://example.com/flake?dir=subdir", Dir: "subdir"},
	}

	for ref, want := range cases {
		t.Run(ref, func(t *testing.T) {
			got, err := ParseFlakeRef(ref)
			if diff := cmp.Diff(want, got); diff != "" {
				if err != nil {
					t.Errorf("got error: %s", err)
				}
				t.Errorf("wrong flakeref (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFlakeRefString(t *testing.T) {
	cases := map[FlakeRef]string{
		{}: "",

		// Path references.
		{Type: "path", Path: "."}:                "path:.",
		{Type: "path", Path: "./"}:               "path:.",
		{Type: "path", Path: "./flake"}:          "path:flake",
		{Type: "path", Path: "./relative/flake"}: "path:relative/flake",
		{Type: "path", Path: "/"}:                "path:/",
		{Type: "path", Path: "/flake"}:           "path:/flake",
		{Type: "path", Path: "/absolute/flake"}:  "path:/absolute/flake",

		// Path references with escapes.
		{Type: "path", Path: "%"}:                 "path:%25",
		{Type: "path", Path: "/%2F"}:              "path:/%252F",
		{Type: "path", Path: "./Ûñî©ôδ€/flake\n"}: "path:%C3%9B%C3%B1%C3%AE%C2%A9%C3%B4%CE%B4%E2%82%AC/flake%0A",
		{Type: "path", Path: "/Ûñî©ôδ€/flake\n"}:  "path:/%C3%9B%C3%B1%C3%AE%C2%A9%C3%B4%CE%B4%E2%82%AC/flake%0A",

		// Indirect references.
		{Type: "indirect", ID: "indirect"}:                                                              "flake:indirect",
		{Type: "indirect", ID: "indirect", Dir: "sub/dir"}:                                              "flake:indirect?dir=sub%2Fdir",
		{Type: "indirect", ID: "indirect", Ref: "ref"}:                                                  "flake:indirect/ref",
		{Type: "indirect", ID: "indirect", Ref: "my/ref"}:                                               "flake:indirect/my%2Fref",
		{Type: "indirect", ID: "indirect", Rev: "5233fd2ba76a3accb5aaa999c00509a11fd0793c"}:             "flake:indirect/5233fd2ba76a3accb5aaa999c00509a11fd0793c",
		{Type: "indirect", ID: "indirect", Ref: "ref", Rev: "5233fd2ba76a3accb5aaa999c00509a11fd0793c"}: "flake:indirect/ref/5233fd2ba76a3accb5aaa999c00509a11fd0793c",

		// GitHub references.
		{Type: "github", Owner: "NixOS", Repo: "nix"}:                                                  "github:NixOS/nix",
		{Type: "github", Owner: "NixOS", Repo: "nix", Ref: "v1.2.3"}:                                   "github:NixOS/nix/v1.2.3",
		{Type: "github", Owner: "NixOS", Repo: "nix", Ref: "my/ref"}:                                   "github:NixOS/nix/my%2Fref",
		{Type: "github", Owner: "NixOS", Repo: "nix", Ref: "5233fd2ba76a3accb5aaa999c00509a11fd0793c"}: "github:NixOS/nix/5233fd2ba76a3accb5aaa999c00509a11fd0793c",
		{Type: "github", Owner: "NixOS", Repo: "nix", Ref: "5233fd2bb76a3accb5aaa999c00509a11fd0793z"}: "github:NixOS/nix/5233fd2bb76a3accb5aaa999c00509a11fd0793z",
		{Type: "github", Owner: "NixOS", Repo: "nix", Dir: "sub/dir"}:                                  "github:NixOS/nix?dir=sub%2Fdir",
		{Type: "github", Owner: "NixOS", Repo: "nix", Dir: "sub/dir", Host: "example.com"}:             "github:NixOS/nix?dir=sub%2Fdir&host=example.com",

		// Git references.
		{Type: "git", URL: "git://example.com/repo/flake"}:     "git://example.com/repo/flake",
		{Type: "git", URL: "https://example.com/repo/flake"}:   "git+https://example.com/repo/flake",
		{Type: "git", URL: "ssh://git@example.com/repo/flake"}: "git+ssh://git@example.com/repo/flake",
		{Type: "git", URL: "git:/repo/flake"}:                  "git:/repo/flake",
		{Type: "git", URL: "file:///repo/flake"}:               "git+file:///repo/flake",
		{Type: "git", URL: "ssh://git@example.com/repo/flake", Ref: "my/ref", Rev: "e486d8d40e626a20e06d792db8cc5ac5aba9a5b4"}:                               "git+ssh://git@example.com/repo/flake?ref=my%2Fref&rev=e486d8d40e626a20e06d792db8cc5ac5aba9a5b4",
		{Type: "git", URL: "ssh://git@example.com/repo/flake?dir=sub%2Fdir", Ref: "my/ref", Rev: "e486d8d40e626a20e06d792db8cc5ac5aba9a5b4", Dir: "sub/dir"}: "git+ssh://git@example.com/repo/flake?dir=sub%2Fdir&ref=my%2Fref&rev=e486d8d40e626a20e06d792db8cc5ac5aba9a5b4",
		{Type: "git", URL: "git:repo/flake?dir=sub%2Fdir", Ref: "my/ref", Rev: "e486d8d40e626a20e06d792db8cc5ac5aba9a5b4", Dir: "sub/dir"}:                   "git:repo/flake?dir=sub%2Fdir&ref=my%2Fref&rev=e486d8d40e626a20e06d792db8cc5ac5aba9a5b4",

		// Tarball references.
		{Type: "tarball", URL: "http://example.com/flake"}:                  "tarball+http://example.com/flake",
		{Type: "tarball", URL: "https://example.com/flake"}:                 "tarball+https://example.com/flake",
		{Type: "tarball", URL: "https://example.com/flake", Dir: "sub/dir"}: "tarball+https://example.com/flake?dir=sub%2Fdir",
		{Type: "tarball", URL: "file:///home/flake"}:                        "tarball+file:///home/flake",

		// File URL references.
		{Type: "file", URL: "file:///flake"}:                                              "file+file:///flake",
		{Type: "file", URL: "http://example.com/flake"}:                                   "file+http://example.com/flake",
		{Type: "file", URL: "http://example.com/flake.git"}:                               "file+http://example.com/flake.git",
		{Type: "file", URL: "http://example.com/flake.tar?dir=sub%2Fdir", Dir: "sub/dir"}: "file+http://example.com/flake.tar?dir=sub%2Fdir",
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
	cases := map[string]FlakeInstallable{
		// Empty string is not a valid installable.
		"": {},

		// Not a path and not a valid URL.
		"://bad/url": {},

		".":             {Ref: FlakeRef{Type: "path", Path: "."}},
		".#app":         {AttrPath: "app", Ref: FlakeRef{Type: "path", Path: "."}},
		".#app^out":     {AttrPath: "app", Outputs: "out", Ref: FlakeRef{Type: "path", Path: "."}},
		".#app^out,lib": {AttrPath: "app", Outputs: "lib,out", Ref: FlakeRef{Type: "path", Path: "."}},
		".#app^*":       {AttrPath: "app", Outputs: "*", Ref: FlakeRef{Type: "path", Path: "."}},
		".^*":           {Outputs: "*", Ref: FlakeRef{Type: "path", Path: "."}},

		"./flake":             {Ref: FlakeRef{Type: "path", Path: "./flake"}},
		"./flake#app":         {AttrPath: "app", Ref: FlakeRef{Type: "path", Path: "./flake"}},
		"./flake#app^out":     {AttrPath: "app", Outputs: "out", Ref: FlakeRef{Type: "path", Path: "./flake"}},
		"./flake#app^out,lib": {AttrPath: "app", Outputs: "lib,out", Ref: FlakeRef{Type: "path", Path: "./flake"}},
		"./flake^out":         {Outputs: "out", Ref: FlakeRef{Type: "path", Path: "./flake"}},

		"indirect":            {Ref: FlakeRef{Type: "indirect", ID: "indirect"}},
		"nixpkgs#app":         {AttrPath: "app", Ref: FlakeRef{Type: "indirect", ID: "nixpkgs"}},
		"nixpkgs#app^out":     {AttrPath: "app", Outputs: "out", Ref: FlakeRef{Type: "indirect", ID: "nixpkgs"}},
		"nixpkgs#app^out,lib": {AttrPath: "app", Outputs: "lib,out", Ref: FlakeRef{Type: "indirect", ID: "nixpkgs"}},
		"nixpkgs^out":         {Outputs: "out", Ref: FlakeRef{Type: "indirect", ID: "nixpkgs"}},

		"%23#app":                        {AttrPath: "app", Ref: FlakeRef{Type: "indirect", ID: "#"}},
		"./%23#app":                      {AttrPath: "app", Ref: FlakeRef{Type: "path", Path: "./#"}},
		"/%23#app":                       {AttrPath: "app", Ref: FlakeRef{Type: "path", Path: "/#"}},
		"path:/%23#app":                  {AttrPath: "app", Ref: FlakeRef{Type: "path", Path: "/#"}},
		"http://example.com/%23.tar#app": {AttrPath: "app", Ref: FlakeRef{Type: "tarball", URL: "http://example.com/%23.tar#app"}},
	}

	for installable, want := range cases {
		t.Run(installable, func(t *testing.T) {
			got, err := ParseFlakeInstallable(installable)
			if diff := cmp.Diff(want, got, cmpopts.IgnoreUnexported(FlakeRef{}, FlakeInstallable{})); diff != "" {
				if err != nil {
					t.Errorf("got error: %s", err)
				}
				t.Errorf("wrong installable (-want +got):\n%s", diff)
			}
			if err != nil {
				return
			}
			if installable != got.String() {
				t.Errorf("got.String() = %q != %q", got, installable)
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
	buildQueryString(elems...)
}
