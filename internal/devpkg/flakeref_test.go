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
		"flake:indirect/my%20ref": {Type: "indirect", ID: "indirect", Ref: "my ref"},
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

		// Interpret opaques with invalid escapes literally.
		"path:./relative/my%flake": {Type: "path", Path: "./relative/my%flake"},
		"flake:indirect/my%ref":    {Type: "indirect", ID: "indirect", Ref: "my%ref"},
		"github:NixOS/nix/my%ref":  {Type: "github", Owner: "NixOS", Repo: "nix", Ref: "my%ref"},
	}

	for ref, want := range cases {
		t.Run(ref, func(t *testing.T) {
			got, err := ParseFlakeRef(ref)
			if diff := cmp.Diff(want, got, cmpopts.IgnoreUnexported(FlakeRef{})); diff != "" {
				if err != nil {
					t.Errorf("got error: %s", err)
				}
				t.Errorf("wrong flakeref (-want +got):\n%s", diff)
			}
			if err != nil {
				return
			}
			if ref != got.String() {
				t.Errorf("got.String() = %q != %q", got, ref)
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
		".#app^out":     {AttrPath: "app", Outputs: []string{"out"}, Ref: FlakeRef{Type: "path", Path: "."}},
		".#app^out,lib": {AttrPath: "app", Outputs: []string{"out", "lib"}, Ref: FlakeRef{Type: "path", Path: "."}},
		".#app^*":       {AttrPath: "app", Outputs: []string{"*"}, Ref: FlakeRef{Type: "path", Path: "."}},
		".^*":           {Outputs: []string{"*"}, Ref: FlakeRef{Type: "path", Path: "."}},

		"./flake":             {Ref: FlakeRef{Type: "path", Path: "./flake"}},
		"./flake#app":         {AttrPath: "app", Ref: FlakeRef{Type: "path", Path: "./flake"}},
		"./flake#app^out":     {AttrPath: "app", Outputs: []string{"out"}, Ref: FlakeRef{Type: "path", Path: "./flake"}},
		"./flake#app^out,lib": {AttrPath: "app", Outputs: []string{"out", "lib"}, Ref: FlakeRef{Type: "path", Path: "./flake"}},
		"./flake^out":         {Outputs: []string{"out"}, Ref: FlakeRef{Type: "path", Path: "./flake"}},

		"indirect":            {Ref: FlakeRef{Type: "indirect", ID: "indirect"}},
		"nixpkgs#app":         {AttrPath: "app", Ref: FlakeRef{Type: "indirect", ID: "nixpkgs"}},
		"nixpkgs#app^out":     {AttrPath: "app", Outputs: []string{"out"}, Ref: FlakeRef{Type: "indirect", ID: "nixpkgs"}},
		"nixpkgs#app^out,lib": {AttrPath: "app", Outputs: []string{"out", "lib"}, Ref: FlakeRef{Type: "indirect", ID: "nixpkgs"}},
		"nixpkgs^out":         {Outputs: []string{"out"}, Ref: FlakeRef{Type: "indirect", ID: "nixpkgs"}},

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

func TestFlakeInstallableDefaultOutputs(t *testing.T) {
	install := FlakeInstallable{Outputs: nil}
	if !install.DefaultOutputs() {
		t.Errorf("DefaultOutputs() = false for nil outputs slice, want true")
	}

	install = FlakeInstallable{Outputs: []string{}}
	if !install.DefaultOutputs() {
		t.Errorf("DefaultOutputs() = false for empty outputs slice, want true")
	}

	install = FlakeInstallable{Outputs: []string{"out"}}
	if install.DefaultOutputs() {
		t.Errorf("DefaultOutputs() = true for %v, want false", install.Outputs)
	}
}

func TestFlakeInstallableAllOutputs(t *testing.T) {
	install := FlakeInstallable{Outputs: []string{"*"}}
	if !install.AllOutputs() {
		t.Errorf("AllOutputs() = false for %v, want true", install.Outputs)
	}
	install = FlakeInstallable{Outputs: []string{"out", "*"}}
	if !install.AllOutputs() {
		t.Errorf("AllOutputs() = false for %v, want true", install.Outputs)
	}
	install = FlakeInstallable{Outputs: nil}
	if install.AllOutputs() {
		t.Errorf("AllOutputs() = true for nil outputs slice, want false")
	}
	install = FlakeInstallable{Outputs: []string{}}
	if install.AllOutputs() {
		t.Errorf("AllOutputs() = true for empty outputs slice, want false")
	}
}
