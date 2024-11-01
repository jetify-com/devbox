// Package flake parses and formats Nix flake references.
package flake

import (
	"fmt"
	"net/url"
	"path"
	"slices"
	"strings"

	"go.jetpack.io/devbox/internal/redact"
)

// Flake reference types supported by this package.
const (
	TypeIndirect  = "indirect"
	TypePath      = "path"
	TypeFile      = "file"
	TypeSSH       = "ssh"
	TypeGitHub    = "github"
	TypeGitLab    = "gitlab"
	TypeBitBucket = "bitbucket"
	TypeTarball   = "tarball"
	TypeBuiltin   = "builtin"
)

// Ref is a parsed Nix flake reference. A flake reference is a subset of the
// Nix CLI "installable" syntax. An [Installable] may specify an attribute path
// and derivation outputs with a flake reference using the '#' and '^'
// characters. For example, the string "nixpkgs" and "./flake" are valid flake
// references, but "nixpkgs#hello" and "./flake#app^bin,dev" are not.
//
// The JSON encoding of Ref corresponds to the exploded attribute set form of
// the flake reference in Nix. See the [Nix manual] for details on flake
// references.
//
// [Nix manual]: https://nixos.org/manual/nix/unstable/command-ref/new-cli/nix3-flake
type Ref struct {
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
	// flake ref.
	URL string `json:"url,omitempty"`

	// Port of the server git server, to support privately hosted git servers or tunnels
	Port int32 `json:port,omitempty`
}

// ParseRef parses a raw flake reference. Nix supports a variety of flake ref
// formats, and isn't entirely consistent about how it parses them. ParseRef
// attempts to mimic how Nix parses flake refs on the command line. The raw ref
// can be one of the following:
//
//   - Indirect reference such as "nixpkgs" or "nixpkgs/unstable".
//   - Path-like reference such as "./flake" or "/path/to/flake". They must
//     start with a '.' or '/' and not contain a '#' or '?'.
//   - URL-like reference which must be a valid URL with any special characters
//     encoded. The scheme can be any valid flake ref type except for mercurial,
//     gitlab, and sourcehut.
//
// ParseRef does not guarantee that a parsed flake ref is valid or that an
// error indicates an invalid flake ref. Use the "nix flake metadata" command or
// the builtins.parseFlakeRef Nix function to validate a flake ref.
func ParseRef(ref string) (Ref, error) {
	if ref == "" {
		return Ref{}, redact.Errorf("empty flake reference")
	}

	// Handle path-style references first.
	parsed := Ref{}
	if ref[0] == '.' || ref[0] == '/' {
		if strings.ContainsAny(ref, "?#") {
			// The Nix CLI does seem to allow paths with a '?'
			// (contrary to the manual) but ignores everything that
			// comes after it. This is a bit surprising, so we just
			// don't allow it at all.
			return Ref{}, redact.Errorf("path-style flake reference %q contains a '?' or '#'", ref)
		}
		parsed.Type = TypePath
		parsed.Path = ref
		return parsed, nil
	}
	parsed, fragment, err := parseURLRef(ref)
	if fragment != "" {
		return Ref{}, redact.Errorf("flake reference %q contains a URL fragment", ref)
	}
	return parsed, err
}

func parseURLRef(ref string) (parsed Ref, fragment string, err error) {
	// A good way to test how Nix parses a flake reference is to run:
	//
	// 	nix eval --json --expr 'builtins.parseFlakeRef "ref"' | jq
	refURL, err := url.Parse(ref)
	if err != nil {
		return Ref{}, "", redact.Errorf("parse flake reference as URL: %v", err)
	}

	// ensure that the fragment is excluded from the parsed URL
	// since those are not valid in flake references.
	fragment = refURL.Fragment
	refURL.Fragment = ""

	switch refURL.Scheme {
	case "", "flake":
		// [flake:]<flake-id>(/<rev-or-ref>(/rev)?)?

		parsed.Type = TypeIndirect
		split, err := splitPathOrOpaque(refURL, -1)
		if err != nil {
			return Ref{}, "", redact.Errorf("parse flake reference URL path: %v", err)
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

		parsed.Type = TypePath
		if refURL.Path == "" {
			parsed.Path, err = url.PathUnescape(refURL.Opaque)
			if err != nil {
				return Ref{}, "", err
			}
		} else {
			parsed.Path = refURL.Path
		}
	case "http", "https", "file":
		if isArchive(refURL.Path) {
			parsed.Type = TypeTarball
		} else {
			parsed.Type = TypeFile
		}
		parsed.Dir = refURL.Query().Get("dir")
		parsed.URL = refURL.String()
	case "tarball+http", "tarball+https", "tarball+file":
		parsed.Type = TypeTarball
		parsed.Dir = refURL.Query().Get("dir")

		refURL.Scheme = refURL.Scheme[8:] // remove tarball+
		parsed.URL = refURL.String()
	case "file+http", "file+https", "file+file":
		parsed.Type = TypeFile
		parsed.Dir = refURL.Query().Get("dir")

		refURL.Scheme = refURL.Scheme[5:] // remove file+
		parsed.URL = refURL.String()
	case "git", "git+http", "git+https", "git+ssh", "git+git", "git+file":
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

		if strings.HasPrefix(refURL.Scheme, TypeSSH) {
			parsed.Type = TypeSSH
		} else if strings.HasPrefix(refURL.Scheme, TypeFile) {
			parsed.Type = TypeFile
		}

		parsed.URL = refURL.String()

		if err := parseGitRef(refURL, &parsed); err != nil {
			return Ref{}, "", err
		}
	case "bitbucket":
		parsed.Type = TypeBitBucket
		if err := parseGitRef(refURL, &parsed); err != nil {
			return Ref{}, "", err
		}
	case "gitlab":
		parsed.Type = TypeGitLab
		if err := parseGitRef(refURL, &parsed); err != nil {
			return Ref{}, "", err
		}
	case "github":
		parsed.Type = TypeGitHub
		if err := parseGitRef(refURL, &parsed); err != nil {
			return Ref{}, "", err
		}
	default:
		return Ref{}, "", redact.Errorf("unsupported flake reference URL scheme: %s", redact.Safe(refURL.Scheme))
	}

	return parsed, fragment, nil
}

func parseGitRef(refURL *url.URL, parsed *Ref) error {
	// github:<owner>/<repo>(/<rev-or-ref>)?(\?<params>)?

	parsed.Type = TypeGitHub

	// Only split up to 3 times (owner, repo, ref/rev) so that we handle
	// refs that have slashes in them. For example,
	// "github:jetify-com/devbox/gcurtis/flakeref" parses as "gcurtis/flakeref".
	split, err := splitPathOrOpaque(refURL, 3)
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
	parsed.Dir = refURL.Query().Get("dir")
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
	parsed.Dir = refURL.Query().Get("dir")
	return nil
}

// String encodes the flake reference as a URL-like string. It normalizes the
// result such that if two Ref values are equal, then their strings will also be
// equal.
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
func (r Ref) String() string {
	switch r.Type {

	case TypeFile:
		if r.URL == "" {
			return ""
		}
		return "file+" + r.URL
	case TypeSSH:
		base := fmt.Sprintf("git+ssh://git@%s", r.Host)
		if r.Port > 0 {
			base = fmt.Sprintf("%s:%d", base, r.Port)
		}

		queryParams := url.Values{}

		if r.Rev != "" {
			queryParams.Add("rev", r.Rev)
		}
		if r.Ref != "" {
			queryParams.Add("ref", r.Ref)
		}
		if r.Dir != "" {
			queryParams.Add("dir", r.Dir)
		}

		return fmt.Sprintf("%s/%s/%s?%s", base, r.Owner, r.Repo, queryParams.Encode())

	case TypeGitLab, TypeBitBucket, TypeGitHub:
		if r.Owner == "" || r.Repo == "" {
			return ""
		}

		scheme := "github" // using as default
		if r.Type == TypeGitLab {
			scheme = "gitlab"
		}
		if r.Type == TypeBitBucket {
			scheme = "bitbucket"
		}

		url := &url.URL{
			Scheme:   scheme,
			Opaque:   buildEscapedPath(r.Owner, r.Repo, r.Rev, r.Ref),
			RawQuery: buildQueryString("host", r.Host, "dir", r.Dir),
		}

		return url.String()

	case TypeIndirect:
		if r.ID == "" {
			return ""
		}
		url := &url.URL{
			Scheme:   "flake",
			Opaque:   buildEscapedPath(r.ID, r.Ref, r.Rev),
			RawQuery: buildQueryString("dir", r.Dir),
		}
		return url.String()
	case TypePath:
		if r.Path == "" {
			return ""
		}
		r.Path = path.Clean(r.Path)
		url := &url.URL{
			Scheme: "path",
			Opaque: buildEscapedPath(strings.Split(r.Path, "/")...),
		}

		// Add the / prefix back if strings.Split removed it.
		if r.Path[0] == '/' {
			url.Opaque = "/" + url.Opaque
		} else if r.Path == "." {
			url.Opaque = "."
		}
		return url.String()
	case TypeTarball:
		if r.URL == "" {
			return ""
		}
		if !strings.HasPrefix(r.URL, "tarball") {
			r.URL = "tarball+" + r.URL
		}
		if r.Dir == "" {
			return r.URL
		}

		url, err := url.Parse(r.URL)
		if err != nil {
			// This should be rare and only happen if the caller
			// messed with the parsed URL.
			return ""
		}
		url.RawQuery = buildQueryString("dir", r.Dir)
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
	// As documented under the tarball type:
	// https://nixos.org/manual/nix/unstable/command-ref/new-cli/nix3-flake#types
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
// The count limits the number of substrings per [strings.SplitN]

// TODO git rid of this
func splitPathOrOpaque(u *url.URL, n int) ([]string, error) {
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
	split := strings.SplitN(upath, "/", n)
	for i := range split {
		split[i], err = url.PathUnescape(split[i])
		if err != nil {
			return nil, err
		}
	}
	return split, nil
}

// TODO maybe use this?
func splitRepoString(repo string, n int) ([]string, error) {
	repo = strings.TrimSpace(repo)

	if repo == "" {
		return nil, nil
	}

	// We don't want an empty element if the path is rooted.
	if repo[0] == '/' {
		repo = repo[1:]
	}
	repo = path.Clean(repo)

	var err error
	split := strings.SplitN(repo, "/", n)
	for i := range split {
		split[i], err = url.PathUnescape(split[i])
		if err != nil {
			return nil, err
		}
	}
	return split, nil
}

// buildEscapedPath escapes and joins path elements for a URL flake ref. The
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

// Special values for [Installable].Outputs.
const (
	// DefaultOutputs specifies that the package-defined default outputs
	// should be installed.
	DefaultOutputs = ""

	// AllOutputs specifies that all package outputs should be installed.
	AllOutputs = "*"
)

// Installable is a Nix command line argument that specifies how to install a
// flake. It can be a plain flake reference, or a flake reference with an
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
// in Nix expressions. See [Ref] and the [Nix manual] for details on the
// differences between flake references and installables.
//
// [Nix manual]: https://nixos.org/manual/nix/unstable/command-ref/new-cli/nix#installables
type Installable struct {
	// Ref is the flake reference portion of the installable.
	Ref Ref `json:"ref,omitempty"`

	// AttrPath is an attribute path of the flake, encoded as a URL
	// fragment.
	AttrPath string `json:"attr_path,omitempty"`

	// Outputs is the installable's output spec, which is a comma-separated
	// list of package outputs to install. The outputs spec is anything
	// after the last caret '^' in an installable. Unlike the
	// attribute path, output specs are not URL-encoded.
	//
	// The special values DefaultOutputs ("") and AllOutputs ("*") specify
	// the default set of package outputs and all package outputs,
	// respectively.
	//
	// ParseInstallable cleans the list of outputs by removing empty
	// elements and sorting the results. Lists containing a "*" are
	// simplified to a single "*".
	Outputs string `json:"outputs,omitempty"`
}

// ParseInstallable parses a flake installable. The raw string must contain
// a valid flake reference parsable by [ParseRef], optionally followed by an
// #attrpath and/or an ^output.
func ParseInstallable(raw string) (Installable, error) {
	if raw == "" {
		return Installable{}, redact.Errorf("empty flake installable")
	}

	// The output spec must be parsed and removed first, otherwise it will
	// be parsed as part of the flake ref's URL fragment.
	install := Installable{}
	raw, install.Outputs = splitOutputSpec(raw)
	install.Outputs = strings.Join(install.SplitOutputs(), ",") // clean the outputs

	// Interpret installables with path-style flake refs as URLs to extract
	// the attribute path (fragment). This means that path-style flake refs
	// cannot point to files with a '#' or '?' in their name, since those
	// would be parsed as the URL fragment or query string. This mimic's
	// Nix's CLI behavior.
	if raw[0] == '.' || raw[0] == '/' {
		raw = "path:" + raw
	}

	var err error
	install.Ref, install.AttrPath, err = parseURLRef(raw)
	if err != nil {
		return Installable{}, err
	}
	return install, nil
}

// SplitOutputs splits and sorts the comma-separated list of outputs. It skips
// any empty outputs. If one or more of the outputs is a "*", then the result
// will be a slice with a single "*" element.
func (fi Installable) SplitOutputs() []string {
	if fi.Outputs == "" {
		return []string{}
	}

	split := strings.Split(fi.Outputs, ",")
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
// documentation. The same normalization rules from [Ref.String] still apply.
func (fi Installable) String() string {
	str := fi.Ref.String()
	if str == "" {
		return ""
	}
	if fi.AttrPath != "" {
		url, err := url.Parse(str)
		if err != nil {
			// This should never happen. Even an empty string is a
			// valid URL.
			panic("invalid URL from Ref.String: " + str)
		}
		url.Fragment = fi.AttrPath
		str = url.String()
	}
	if fi.Outputs != "" {
		clean := strings.Join(fi.SplitOutputs(), ",")
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
