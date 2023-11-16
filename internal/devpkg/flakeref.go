package devpkg

import (
	"net/url"
	"path"
	"slices"
	"strings"

	"go.jetpack.io/devbox/internal/redact"
)

const (
	FlakeTypeIndirect = "indirect"
	FlakeTypePath     = "path"
	FlakeTypeFile     = "file"
	FlakeTypeGit      = "git"
	FlakeTypeGitHub   = "github"
	FlakeTypeTarball  = "tarball"
)

// FlakeRef is a parsed Nix flake reference. A flake reference is a subset of
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

	// Dir is non-empty when the directory containing the flake.nix file is
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
	parsed := FlakeRef{}
	if ref[0] == '.' || ref[0] == '/' {
		if strings.ContainsAny(ref, "?#") {
			// The Nix CLI does seem to allow paths with a '?'
			// (contrary to the manual) but ignores everything that
			// comes after it. This is a bit surprising, so we just
			// don't allow it at all.
			return FlakeRef{}, redact.Errorf("path-style flake reference %q contains a '?' or '#'", ref)
		}
		parsed.Type = FlakeTypePath
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
	refURL, err := url.Parse(ref)
	if err != nil {
		return FlakeRef{}, "", redact.Errorf("parse flake reference as URL: %v", err)
	}

	switch refURL.Scheme {
	case "", "flake":
		// [flake:]<flake-id>(/<rev-or-ref>(/rev)?)?

		parsed.Type = FlakeTypeIndirect
		split, err := splitPathOrOpaque(refURL)
		if err != nil {
			return FlakeRef{}, "", redact.Errorf("parse flake reference URL path: %v", err)
		}
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

		parsed.Type = FlakeTypePath
		if refURL.Path == "" {
			parsed.Path, err = url.PathUnescape(refURL.Opaque)
			if err != nil {
				return FlakeRef{}, "", err
			}
		} else {
			parsed.Path = refURL.Path
		}
	case "http", "https", "file":
		if isArchive(refURL.Path) {
			parsed.Type = FlakeTypeTarball
		} else {
			parsed.Type = FlakeTypeFile
		}
		parsed.Dir = refURL.Query().Get("dir")
		parsed.URL = refURL.String()
	case "tarball+http", "tarball+https", "tarball+file":
		parsed.Type = FlakeTypeTarball
		parsed.Dir = refURL.Query().Get("dir")

		refURL.Scheme = refURL.Scheme[8:] // remove tarball+
		parsed.URL = refURL.String()
	case "file+http", "file+https", "file+file":
		parsed.Type = FlakeTypeFile
		parsed.Dir = refURL.Query().Get("dir")

		refURL.Scheme = refURL.Scheme[5:] // remove file+
		parsed.URL = refURL.String()
	case "git", "git+http", "git+https", "git+ssh", "git+git", "git+file":
		parsed.Type = FlakeTypeGit
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

	parsed.Type = FlakeTypeGitHub
	split, err := splitPathOrOpaque(refURL)
	if err != nil {
		return err
	}
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

// String encodes the flake reference as a URL-like string. It normalizes
// the result such that if two FlakeRef values are equal, then their strings
// will also be equal.
//
// There are multiple ways to encode a flake reference. String uses the
// following rules to normalize the result:
//
//   - the URL-like form is always used, even for paths and indirects.
//   - the scheme is always present, even if it's optional.
//   - paths are cleaned per path.Clean.
//   - fields that can be put in either the path or the query string are always
//     put in the path.
//   - query parameters are sorted by key.
//
// If f is missing a type or has any invalid fields, String returns an empty
// string.
func (f FlakeRef) String() string {
	switch f.Type {
	case FlakeTypeFile:
		if f.URL == "" {
			return ""
		}
		return "file+" + f.URL
	case FlakeTypeGit:
		if f.URL == "" {
			return ""
		}
		if !strings.HasPrefix(f.URL, "git") {
			f.URL = "git+" + f.URL
		}

		// Nix removes "ref" and "rev" from the query string
		// (but not other parameters) after parsing. If they're empty,
		// we can skip parsing the URL. Otherwise, we need to add them
		// back.
		if f.Ref == "" && f.Rev == "" {
			return f.URL
		}
		url, err := url.Parse(f.URL)
		if err != nil {
			// This should be rare and only happen if the caller
			// messed with the parsed URL.
			return ""
		}
		url.RawQuery = buildQueryString("ref", f.Ref, "rev", f.Rev, "dir", f.Dir)
		return url.String()
	case FlakeTypeGitHub:
		if f.Owner == "" || f.Repo == "" {
			return ""
		}
		url := &url.URL{
			Scheme:   "github",
			Opaque:   buildEscapedPath(f.Owner, f.Repo, f.Rev, f.Ref),
			RawQuery: buildQueryString("host", f.Host, "dir", f.Dir),
		}
		return url.String()
	case FlakeTypeIndirect:
		if f.ID == "" {
			return ""
		}
		url := &url.URL{
			Scheme:   "flake",
			Opaque:   buildEscapedPath(f.ID, f.Ref, f.Rev),
			RawQuery: buildQueryString("dir", f.Dir),
		}
		return url.String()
	case FlakeTypePath:
		if f.Path == "" {
			return ""
		}
		f.Path = path.Clean(f.Path)
		url := &url.URL{
			Scheme: "path",
			Opaque: buildEscapedPath(strings.Split(f.Path, "/")...),
		}

		// Add the / prefix back if strings.Split removed it.
		if f.Path[0] == '/' {
			url.Opaque = "/" + url.Opaque
		} else if f.Path == "." {
			url.Opaque = "."
		}
		return url.String()
	case FlakeTypeTarball:
		if f.URL == "" {
			return ""
		}
		if !strings.HasPrefix(f.URL, "tarball") {
			f.URL = "tarball+" + f.URL
		}
		if f.Dir == "" {
			return f.URL
		}

		url, err := url.Parse(f.URL)
		if err != nil {
			// This should be rare and only happen if the caller
			// messed with the parsed URL.
			return ""
		}
		url.RawQuery = buildQueryString("dir", f.Dir)
		return url.String()
	default:
		return ""
	}
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

// splitPathOrOpaque splits a URL path by '/'. If the path is empty, it splits
// the opaque instead. Splitting happens before unescaping the path or opaque,
// ensuring that path elements with an encoded '/' (%2F) are not split.
// For example, "/dir/file%2Fname" becomes the elements "dir" and "file/name".
func splitPathOrOpaque(u *url.URL) ([]string, error) {
	upath := u.EscapedPath()
	if upath == "" {
		upath = u.Opaque
	}
	upath = strings.TrimSpace(upath)
	if upath == "" {
		return nil, nil
	}

	// We don't want an empty element if the path is rooted.
	if upath[0] == '/' {
		upath = upath[1:]
	}
	upath = path.Clean(upath)

	var err error
	split := strings.Split(upath, "/")
	for i := range split {
		split[i], err = url.PathUnescape(split[i])
		if err != nil {
			return nil, err
		}
	}
	return split, nil
}

// buildEscapedPath escapes and joins path elements for a URL flakeref. The
// resulting path is cleaned according to url.JoinPath.
func buildEscapedPath(elem ...string) string {
	for i := range elem {
		elem[i] = url.PathEscape(elem[i])
	}
	u := &url.URL{}
	return u.JoinPath(elem...).String()
}

// buildQueryString builds a URL query string from a list of key-value string
// pairs, omitting any keys with empty values.
func buildQueryString(keyval ...string) string {
	if len(keyval)%2 != 0 {
		panic("buildQueryString: odd number of key-value pairs")
	}

	query := make(url.Values, len(keyval)/2)
	for i := 0; i < len(keyval); i += 2 {
		k, v := keyval[i], keyval[i+1]
		if v != "" {
			query.Set(k, v)
		}
	}
	return query.Encode()
}

// Special values for FlakeInstallable.Outputs.
const (
	// DefaultOutputs specifies that the package-defined default outputs
	// should be installed.
	DefaultOutputs = ""

	// AllOutputs specifies that all package outputs should be installed.
	AllOutputs = "*"
)

// FlakeInstallable is a Nix command line argument that specifies how to install
// a flake. It can be a plain flake reference, or a flake reference with an
// attribute path and/or output specification.
//
// Some examples are:
//
//   - "." installs the default attribute from the flake in the current
//     directory.
//   - ".#hello" installs the hello attribute from the flake in the current
//     directory.
//   - "nixpkgs#hello" installs the hello attribute from the nixpkgs flake.
//   - "github:NixOS/nixpkgs/unstable#curl^lib" installs the the lib output of
//     curl attribute from the flake on the nixpkgs unstable branch.
//
// The flake installable syntax is only valid in Nix command line arguments, not
// in Nix expressions. See FlakeRef and the [Nix manual for details on the
// differences between flake references and installables.
//
// [Nix manual]: https://nixos.org/manual/nix/unstable/command-ref/new-cli/nix#installables
type FlakeInstallable struct {
	Ref      FlakeRef
	AttrPath string

	Outputs string
}

// ParseFlakeInstallable parses a flake installable. The string s must contain a
// valid flake reference parsable by ParseFlakeRef, optionally followed by an
// #attrpath and/or an ^output.
func ParseFlakeInstallable(raw string) (FlakeInstallable, error) {
	if raw == "" {
		return FlakeInstallable{}, redact.Errorf("empty flake installable")
	}

	// The output spec must be parsed and removed first, otherwise it will
	// be parsed as part of the flakeref's URL fragment.
	install := FlakeInstallable{}
	raw, install.Outputs = splitOutputSpec(raw)
	install.Outputs = strings.Join(install.SplitOutputs(), ",") // clean the outputs

	// Interpret installables with path-style flakerefs as URLs to extract
	// the attribute path (fragment). This means that path-style flakerefs
	// cannot point to files with a '#' or '?' in their name, since those
	// would be parsed as the URL fragment or query string. This mimic's
	// Nix's CLI behavior.
	if raw[0] == '.' || raw[0] == '/' {
		raw = "path:" + raw
	}

	var err error
	install.Ref, install.AttrPath, err = parseFlakeURLRef(raw)
	if err != nil {
		return FlakeInstallable{}, err
	}
	return install, nil
}

// SplitOutputs splits and sorts the comma-separated list of outputs. It skips
// any empty outputs. If one or more of the outputs is a "*", then the result
// will be a slice with a single "*" element.
func (f FlakeInstallable) SplitOutputs() []string {
	if f.Outputs == "" {
		return []string{}
	}

	split := strings.Split(f.Outputs, ",")
	i := 0
	for _, out := range split {
		// A wildcard takes priority over any other outputs.
		if out == "*" {
			return []string{"*"}
		}
		if out != "" {
			split[i] = out
			i++
		}
	}
	split = split[:i]
	slices.Sort(split)
	return split
}

// String encodes the installable as a Nix command line argument. It normalizes
// the result such that if two installable values are equal, then their strings
// will also be equal.
//
// String always cleans the outputs spec as described by the Outputs field's
// documentation. The same normalization rules from FlakeRef.String still apply.
func (f FlakeInstallable) String() string {
	str := f.Ref.String()
	if str == "" {
		return ""
	}
	if f.AttrPath != "" {
		url, err := url.Parse(str)
		if err != nil {
			// This should never happen. Even an empty string is a
			// valid URL.
			panic("invalid URL from FlakeRef.String: " + str)
		}
		url.Fragment = f.AttrPath
		str = url.String()
	}
	if f.Outputs != "" {
		clean := strings.Join(f.SplitOutputs(), ",")
		if clean != "" {
			str += "^" + clean
		}
	}
	return str
}

// splitOutputSpec cuts a flake installable around the last instance of ^.
func splitOutputSpec(s string) (before, after string) {
	if i := strings.LastIndexByte(s, '^'); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}
