package devpkg

import (
	"net/url"
	"strings"

	"go.jetpack.io/devbox/internal/redact"
)

// FlakeRef is a parsed Nix flake reference. A flake reference is as subset of
// the Nix CLI "installable" syntax. Installables may specify an attribute path
// and derivation outputs with a flake reference using the '#' and '^' characters.
// For example, the string "nixpkgs" and "./flake" are valid flake references,
// but "nixpkgs#hello" and "./flake#app^bin,dev" are not.
//
// See the [Nix manual] for details on flake references.
//
// [Nix manual]: https://nixos.org/manual/nix/unstable/command-ref/new-cli/nix3-flake
type FlakeRef struct {
	// Type is the type of flake reference. Some valid types are "indirect",
	// "path", "file", "git", "tarball", and "github".
	Type string `json:"type,omitempty"`

	// ID is the flake's identifier when Type is "indirect". A common
	// example is nixpkgs.
	ID string `json:"id,omitempty"`

	// Path is the path to the flake directory when Type is "path".
	Path string `json:"path,omitempty"`

	// Owner and repo are the flake repository owner and name when Type is
	// "github".
	Owner string `json:"owner,omitempty"`
	Repo  string `json:"repo,omitempty"`

	// Rev and ref are the git revision (commit hash) and ref
	// (branch or tag) when Type is "github" or "git".
	Rev string `json:"rev,omitempty"`
	Ref string `json:"ref,omitempty"`

	// Dir is non-empty when the directory containinig the flake.nix file is
	// not at the flake root. It corresponds to the optional "dir" query
	// parameter when Type is "github", "git", "tarball", or "file".
	Dir string `json:"dir,omitempty"`

	// Host overrides the default VCS host when Type is "github", such as
	// when referring to a GitHub Enterprise instance. It corresponds to the
	// optional "host" query parameter when Type is "github".
	Host string `json:"host,omitempty"`

	// URL is the URL pointing to the flake when type is "tarball", "file",
	// or "git". Note that the URL is not the same as the raw unparsed
	// flakeref.
	URL string `json:"url,omitempty"`

	// raw stores the original unparsed flakeref string.
	raw string
}

// ParseFlakeRef parses a raw flake reference. Nix supports a variety of
// flakeref formats, and isn't entirely consistent about how it parses them.
// ParseFlakeRef attempts to mimic how Nix parses flakerefs on the command line.
// The raw ref can be one of the following:
//
//   - Indirect reference such as "nixpkgs" or "nixpkgs/unstable".
//   - Path-like reference such as "./flake" or "/path/to/flake". They must
//     start with a '.' or '/' and not contain a '#' or '?'.
//   - URL-like reference which must be a valid URL with any special characters
//     encoded. The scheme can be any valid flakeref type except for mercurial,
//     gitlab, and sourcehut.
//
// ParseFlakeRef does not guarantee that a parsed flakeref is valid or that an
// error indicates an invalid flakeref. Use the "nix flake metadata" command or
// the parseFlakeRef builtin function to validate a flakeref.
func ParseFlakeRef(ref string) (FlakeRef, error) {
	if ref == "" {
		return FlakeRef{}, redact.Errorf("empty flake reference")
	}

	// Handle path-style references first.
	parsed := FlakeRef{raw: ref}
	if ref[0] == '.' || ref[0] == '/' {
		if strings.ContainsAny(ref, "?#") {
			// The Nix CLI does seem to allow paths with a '?'
			// (contrary to the manual) but ignores everything that
			// comes after it. This is a bit surprising, so we just
			// don't allow it at all.
			return FlakeRef{}, redact.Errorf("path-style flake reference %q contains a '?' or '#'", ref)
		}
		parsed.Type = "path"
		parsed.Path = ref
		return parsed, nil
	}
	parsed, _, err := parseFlakeURLRef(ref)
	return parsed, err
}

func parseFlakeURLRef(ref string) (parsed FlakeRef, fragment string, err error) {
	// A good way to test how Nix parses a flake reference is to run:
	//
	// 	nix eval --json --expr 'builtins.parseFlakeRef "ref"' | jq
	parsed.raw = ref
	refURL, err := url.Parse(ref)
	if err != nil {
		return FlakeRef{}, "", redact.Errorf("parse flake reference as URL: %v", err)
	}

	switch refURL.Scheme {
	case "", "flake":
		// [flake:]<flake-id>(/<rev-or-ref>(/rev)?)?

		parsed.Type = "indirect"

		// "indirect" is parsed as a path, "flake:indirect" is parsed as
		// opaque because it has a scheme.
		path := refURL.Path
		if path == "" {
			path, err = url.PathUnescape(refURL.Opaque)
			if err != nil {
				path = refURL.Opaque
			}
		}
		split := strings.SplitN(path, "/", 3)
		parsed.ID = split[0]
		if len(split) > 1 {
			if isGitHash(split[1]) {
				parsed.Rev = split[1]
			} else {
				parsed.Ref = split[1]
			}
		}
		if len(split) > 2 && parsed.Rev == "" {
			parsed.Rev = split[2]
		}
	case "path":
		// [path:]<path>(\?<params)?

		parsed.Type = "path"
		if refURL.Path == "" {
			parsed.Path, err = url.PathUnescape(refURL.Opaque)
			if err != nil {
				parsed.Path = refURL.Opaque
			}
		} else {
			parsed.Path = refURL.Path
		}
	case "http", "https", "file":
		if isArchive(refURL.Path) {
			parsed.Type = "tarball"
		} else {
			parsed.Type = "file"
		}
		parsed.URL = ref
		parsed.Dir = refURL.Query().Get("dir")
	case "tarball+http", "tarball+https", "tarball+file":
		parsed.Type = "tarball"
		parsed.Dir = refURL.Query().Get("dir")
		parsed.URL = ref[8:] // remove tarball+
	case "file+http", "file+https", "file+file":
		parsed.Type = "file"
		parsed.Dir = refURL.Query().Get("dir")
		parsed.URL = ref[5:] // remove file+
	case "git", "git+http", "git+https", "git+ssh", "git+git", "git+file":
		parsed.Type = "git"
		q := refURL.Query()
		parsed.Dir = q.Get("dir")
		parsed.Ref = q.Get("ref")
		parsed.Rev = q.Get("rev")

		// ref and rev get stripped from the query parameters, but dir
		// stays.
		q.Del("ref")
		q.Del("rev")
		refURL.RawQuery = q.Encode()
		if len(refURL.Scheme) > 3 {
			refURL.Scheme = refURL.Scheme[4:] // remove git+
		}
		parsed.URL = refURL.String()
	case "github":
		if err := parseGitHubFlakeRef(refURL, &parsed); err != nil {
			return FlakeRef{}, "", err
		}
	}
	return parsed, refURL.Fragment, nil
}

func parseGitHubFlakeRef(refURL *url.URL, parsed *FlakeRef) error {
	// github:<owner>/<repo>(/<rev-or-ref>)?(\?<params>)?

	parsed.Type = "github"
	path := refURL.Path
	if path == "" {
		var err error
		path, err = url.PathUnescape(refURL.Opaque)
		if err != nil {
			path = refURL.Opaque
		}
	}
	path = strings.TrimPrefix(path, "/")

	split := strings.SplitN(path, "/", 3)
	parsed.Owner = split[0]
	parsed.Repo = split[1]
	if len(split) > 2 {
		if revOrRef := split[2]; isGitHash(revOrRef) {
			parsed.Rev = revOrRef
		} else {
			parsed.Ref = revOrRef
		}
	}

	parsed.Host = refURL.Query().Get("host")
	if qRef := refURL.Query().Get("ref"); qRef != "" {
		if parsed.Rev != "" {
			return redact.Errorf("github flake reference has a ref and a rev")
		}
		if parsed.Ref != "" && qRef != parsed.Ref {
			return redact.Errorf("github flake reference has a ref in the path (%q) and a ref query parameter (%q)", parsed.Ref, qRef)
		}
		parsed.Ref = qRef
	}
	if qRev := refURL.Query().Get("rev"); qRev != "" {
		if parsed.Ref != "" {
			return redact.Errorf("github flake reference has a ref and a rev")
		}
		if parsed.Rev != "" && qRev != parsed.Rev {
			return redact.Errorf("github flake reference has a rev in the path (%q) and a rev query parameter (%q)", parsed.Rev, qRev)
		}
		parsed.Rev = qRev
	}
	return nil
}

// String returns the raw flakeref string as given to ParseFlakeRef.
func (f FlakeRef) String() string {
	return f.raw
}

func isGitHash(s string) bool {
	if len(s) != 40 {
		return false
	}
	for i := range s {
		isDigit := s[i] >= '0' && s[i] <= '9'
		isHexLetter := s[i] >= 'a' && s[i] <= 'f'
		if !isDigit && !isHexLetter {
			return false
		}
	}
	return true
}

func isArchive(path string) bool {
	return strings.HasSuffix(path, ".tar") ||
		strings.HasSuffix(path, ".tar.gz") ||
		strings.HasSuffix(path, ".tgz") ||
		strings.HasSuffix(path, ".tar.xz") ||
		strings.HasSuffix(path, ".tar.zst") ||
		strings.HasSuffix(path, ".tar.bz2") ||
		strings.HasSuffix(path, ".zip")
}
