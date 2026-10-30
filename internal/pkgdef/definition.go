// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

// Package pkgdef implements package definitions: declarative files that
// describe how to install a tool from the upstream vendor's binary
// distribution instead of nixpkgs. See specs/package-definitions.md for the
// format.
//
// The package is a leaf: it must not import devpkg, lock or plugin so that
// those packages can depend on it.
package pkgdef

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/tailscale/hujson"
)

// SchemaVersion is the only value of the "schema" field this version of
// Devbox understands.
const SchemaVersion = 1

// Scheme is the devbox.json package scheme for package definitions,
// e.g. "pkg:./go.pkg.jsonc@1.22".
const Scheme = "pkg"

// Prefix is Scheme followed by a colon.
const Prefix = Scheme + ":"

// Source is the devbox.lock "source" value for packages installed from a
// package definition.
const Source = "pkg-def"

// Version source types.
const (
	VersionsStatic         = "static"
	VersionsHTTPJSON       = "http_json"
	VersionsGitHubReleases = "github_releases"
	VersionsScript         = "script"
)

// Install strategy types.
const (
	InstallArchive = "archive"
	InstallBinary  = "binary"
	InstallManager = "manager"
	InstallScript  = "script"
)

// Checksum source types.
const (
	ChecksumHTTPJSON      = "http_json"
	ChecksumSHA256URL     = "sha256_url"
	ChecksumSHA512URL     = "sha512_url"
	ChecksumGitHubRelease = "github_release"
	ChecksumNone          = "none"
)

// Definition is a parsed package definition file.
type Definition struct {
	Schema      int    `json:"schema"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Homepage    string `json:"homepage,omitempty"`
	License     string `json:"license,omitempty"`

	Platforms []string          `json:"platforms"`
	OSMap     map[string]string `json:"os_map,omitempty"`
	ArchMap   map[string]string `json:"arch_map,omitempty"`

	Versions VersionSources `json:"versions"`
	Install  Install        `json:"install"`

	Bin      []string `json:"bin,omitempty"`
	Binaries []string `json:"binaries,omitempty"`

	// Activation. These have the same schema as plugin.json and are
	// handed to the plugin manager by the caller; pkgdef only validates
	// that they are well-formed JSON.
	Env         map[string]string          `json:"env,omitempty"`
	Shell       map[string]json.RawMessage `json:"shell,omitempty"`
	CreateFiles map[string]string          `json:"create_files,omitempty"`

	Hooks Hooks `json:"hooks,omitempty"`

	// baseDir is the directory of the definition file for local
	// definitions. Relative file:// URLs are resolved against it.
	baseDir string
}

// ResolveURL makes a relative file:// URL ("file://artifacts/x.tar.gz")
// absolute by resolving it against the directory of a local definition.
// Every other URL is returned unchanged.
func (d *Definition) ResolveURL(rawURL string) string {
	p, ok := strings.CutPrefix(rawURL, "file://")
	if !ok || filepath.IsAbs(p) || d.baseDir == "" {
		return rawURL
	}
	return "file://" + filepath.Join(d.baseDir, p)
}

// VersionSources is one version source or a list of them. In JSON it is
// either an object or an array of objects. A list is the union of its
// sources, for packages whose versions are published in more than one place
// (yarn 1.x and yarn 2+ live in different npm packages).
type VersionSources []VersionSource

// UnmarshalJSON accepts a single source object or an array of them.
func (v *VersionSources) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '[' {
		var list []VersionSource
		if err := json.Unmarshal(data, &list); err != nil {
			return err
		}
		*v = list
		return nil
	}
	var single VersionSource
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	*v = VersionSources{single}
	return nil
}

// MarshalJSON writes a single source as an object so that a definition
// round-trips to the shape it was written in.
func (v VersionSources) MarshalJSON() ([]byte, error) {
	if len(v) == 1 {
		return json.Marshal(v[0])
	}
	return json.Marshal([]VersionSource(v))
}

// VersionSource describes how to list the available versions of a package.
type VersionSource struct {
	Type string `json:"type"`

	// static
	Versions []string `json:"versions,omitempty"`

	// http_json. When URL contains {{.Page}}, pages 0, 1, ... are fetched
	// until one returns 404 or no versions.
	URL    string `json:"url,omitempty"`
	Select string `json:"select,omitempty"`

	// github_releases
	Repo        string  `json:"repo,omitempty"`
	StripPrefix *string `json:"strip_prefix,omitempty"`

	// script
	Run string `json:"run,omitempty"`
}

// Install describes how to install one version of a package on one system.
// The zero value of every field means "use the default".
type Install struct {
	Type string `json:"type"`

	// archive and binary: either url (and checksum) or lookup.
	URL      string    `json:"url,omitempty"`
	Checksum *Checksum `json:"checksum,omitempty"`
	Lookup   *Lookup   `json:"lookup,omitempty"`

	// archive
	StripComponents *int `json:"strip_components,omitempty"`

	// binary
	BinName string `json:"bin_name,omitempty"`

	// manager (planned)
	Requires   string            `json:"requires,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	InstallCmd string            `json:"install,omitempty"`
	Locate     string            `json:"locate,omitempty"`

	// script (planned)
	Run string `json:"run,omitempty"`

	// Overrides maps a nix system to a partial Install whose non-zero
	// fields replace the defaults for that system. Overrides may not
	// themselves contain overrides.
	Overrides map[string]*Install `json:"overrides,omitempty"`

	// VersionOverrides are partial Installs for the versions matching
	// their constraint, for vendors that changed how they ship artifacts.
	// The first matching entry applies. An entry may carry its own
	// per-system overrides but no further version overrides.
	VersionOverrides []*VersionOverride `json:"version_overrides,omitempty"`
}

// VersionOverride is an Install that applies to the versions matching
// Versions, a constraint such as "<11" or ">=2 <4". See MatchesConstraint.
type VersionOverride struct {
	Versions string `json:"versions"`
	Install
}

// Lookup finds an artifact in a JSON index: the jq expression in Select is
// evaluated against the document at URL and must produce an object with
// "url" and "checksum" for the requested version and system. A lookup that
// produces nothing means the version is not available for that system.
type Lookup struct {
	URL    string `json:"url"`
	Select string `json:"select"`
}

// Checksum describes where to find the digest of an artifact. Digests may
// be sha256 or sha512, as hex, "sha256:<hex>" or an SRI string
// ("sha512-<base64>").
type Checksum struct {
	Type   string `json:"type"`
	URL    string `json:"url,omitempty"`
	Select string `json:"select,omitempty"`

	// github_release: the asset digest GitHub computes for every release
	// asset. Tag defaults to "v{{.Version}}".
	Repo string `json:"repo,omitempty"`
	Tag  string `json:"tag,omitempty"`
}

// Hooks are shell commands run at specific points of a package's lifecycle.
type Hooks struct {
	PostInstall []string `json:"post_install,omitempty"`
}

var nameRegexp = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

// githubRepoRegexp matches "owner/name" with exactly two segments made of
// the characters GitHub allows, so a repo can never be rewritten into a
// different API path.
var githubRepoRegexp = regexp.MustCompile(`^[A-Za-z0-9-]+/[A-Za-z0-9_.-]+$`)

// knownSystems are the nix systems a definition may list in "platforms".
var knownSystems = []string{
	"aarch64-darwin",
	"x86_64-darwin",
	"aarch64-linux",
	"x86_64-linux",
}

// Parse parses and validates a package definition. The input may contain
// comments and trailing commas (hujson).
func Parse(content []byte) (*Definition, error) {
	std, err := hujson.Standardize(slices.Clone(content))
	if err != nil {
		return nil, fmt.Errorf("package definition is not valid JSON: %w", err)
	}
	def := &Definition{}
	if err := json.Unmarshal(std, def); err != nil {
		return nil, fmt.Errorf("package definition is not valid JSON: %w", err)
	}
	if err := def.Validate(); err != nil {
		return nil, err
	}
	return def, nil
}

// Validate checks the definition for structural errors. Parse calls it; it
// is exported for callers that build a Definition in code.
func (d *Definition) Validate() error {
	var errs []string
	fail := func(format string, args ...any) {
		errs = append(errs, fmt.Sprintf(format, args...))
	}

	if d.Schema != SchemaVersion {
		fail("schema must be %d, got %d", SchemaVersion, d.Schema)
	}
	if d.Name == "" {
		fail("name is required")
	} else if !nameRegexp.MatchString(d.Name) {
		fail("name %q must match %s", d.Name, nameRegexp)
	}
	if len(d.Platforms) == 0 {
		fail("platforms must list at least one system")
	}
	for _, sys := range d.Platforms {
		if !slices.Contains(knownSystems, sys) {
			fail("platforms: unknown system %q (want one of %s)", sys, strings.Join(knownSystems, ", "))
		}
	}
	for _, dir := range d.Bin {
		if dir == "" || strings.HasPrefix(dir, "/") || strings.Contains(dir, "..") {
			fail("bin: %q must be a relative path inside the install root", dir)
		}
	}

	if len(d.Versions) == 0 {
		fail("versions is required")
	}
	for i, src := range d.Versions {
		path := "versions"
		if len(d.Versions) > 1 {
			path = fmt.Sprintf("versions[%d]", i)
		}
		errs = append(errs, src.validate(path)...)
	}

	errs = append(errs, d.Install.validate("install")...)
	errs = append(errs, d.validateSystemOverrides("install", d.Install, d.Install.Overrides)...)
	for i, override := range d.Install.VersionOverrides {
		path := fmt.Sprintf("install.version_overrides[%d]", i)
		if override == nil {
			fail("%s must be an object", path)
			continue
		}
		if err := validateConstraint(override.Versions); err != nil {
			fail("%s.versions: %v", path, err)
		}
		if len(override.VersionOverrides) > 0 {
			fail("%s may not contain version_overrides", path)
		}
		merged := d.Install.merge(&override.Install)
		errs = append(errs, merged.validate(path)...)
		errs = append(errs, d.validateSystemOverrides(path, merged, override.Overrides)...)
	}

	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("invalid package definition %q:\n  - %s", d.Name, strings.Join(errs, "\n  - "))
}

// validateSystemOverrides validates per-system overrides applied on top of
// base, the install they override.
func (d *Definition) validateSystemOverrides(path string, base Install, overrides map[string]*Install) []string {
	var errs []string
	for sys, override := range overrides {
		if !slices.Contains(d.Platforms, sys) {
			errs = append(errs, fmt.Sprintf("%s.overrides: %q is not in platforms", path, sys))
		}
		if override == nil {
			errs = append(errs, fmt.Sprintf("%s.overrides.%s must be an object", path, sys))
			continue
		}
		if len(override.Overrides) > 0 || len(override.VersionOverrides) > 0 {
			errs = append(errs, fmt.Sprintf("%s.overrides.%s may not contain overrides", path, sys))
		}
		merged := base.merge(override)
		errs = append(errs, merged.validate(path+".overrides."+sys)...)
	}
	return errs
}

func (v *VersionSource) validate(path string) []string {
	var errs []string
	switch v.Type {
	case VersionsStatic:
		if len(v.Versions) == 0 {
			errs = append(errs, path+".versions must list at least one version")
		}
	case VersionsHTTPJSON:
		if v.URL == "" {
			errs = append(errs, path+".url is required for type http_json")
		}
		if v.Select == "" {
			errs = append(errs, path+".select is required for type http_json")
		} else if err := validateJQ(v.Select); err != nil {
			errs = append(errs, fmt.Sprintf("%s.select: %v", path, err))
		}
	case VersionsGitHubReleases:
		if !isGitHubRepo(v.Repo) {
			errs = append(errs, path+".repo must be of the form owner/name")
		}
	case VersionsScript:
		if v.Run == "" {
			errs = append(errs, path+".run is required for type script")
		}
	case "":
		errs = append(errs, path+".type is required")
	default:
		errs = append(errs, fmt.Sprintf("%s.type %q is not one of %s, %s, %s, %s",
			path, v.Type, VersionsStatic, VersionsHTTPJSON, VersionsGitHubReleases, VersionsScript))
	}
	return errs
}

func isGitHubRepo(repo string) bool {
	_, name, _ := strings.Cut(repo, "/")
	return githubRepoRegexp.MatchString(repo) && name != "." && name != ".."
}

// validateJQ compiles a jq expression. Expressions that contain template
// actions are only checked once rendered, because an action may expand to
// anything.
func validateJQ(expr string) error {
	if strings.Contains(expr, "{{") {
		return nil
	}
	_, err := compileJQ(expr)
	return err
}

func (in *Install) validate(path string) []string {
	var errs []string
	switch in.Type {
	case InstallArchive, InstallBinary:
		switch {
		case in.URL == "" && in.Lookup == nil:
			errs = append(errs, path+".url or .lookup is required for type "+in.Type)
		case in.URL != "" && in.Lookup != nil:
			errs = append(errs, path+".url and .lookup are mutually exclusive")
		}
		if in.Lookup != nil {
			if in.Lookup.URL == "" || in.Lookup.Select == "" {
				errs = append(errs, path+".lookup.url and .lookup.select are required")
			} else if err := validateJQ(in.Lookup.Select); err != nil {
				errs = append(errs, fmt.Sprintf("%s.lookup.select: %v", path, err))
			}
			if in.Checksum != nil {
				errs = append(errs, path+".checksum is not used with .lookup, which provides the checksum")
			}
		}
		if in.StripComponents != nil && *in.StripComponents < 0 {
			errs = append(errs, path+".strip_components must be >= 0")
		}
		if in.Checksum != nil {
			errs = append(errs, in.Checksum.validate(path+".checksum")...)
		}
	case InstallManager:
		if in.Requires == "" || in.InstallCmd == "" || in.Locate == "" {
			errs = append(errs, path+".requires, .install and .locate are required for type manager")
		}
	case InstallScript:
		if in.Run == "" {
			errs = append(errs, path+".run is required for type script")
		}
	case "":
		errs = append(errs, path+".type is required")
	default:
		errs = append(errs, fmt.Sprintf("%s.type %q is not one of %s, %s, %s, %s",
			path, in.Type, InstallArchive, InstallBinary, InstallManager, InstallScript))
	}
	return errs
}

func (c *Checksum) validate(path string) []string {
	var errs []string
	switch c.Type {
	case ChecksumHTTPJSON:
		if c.URL == "" || c.Select == "" {
			errs = append(errs, path+".url and .select are required for type http_json")
		} else if err := validateJQ(c.Select); err != nil {
			errs = append(errs, fmt.Sprintf("%s.select: %v", path, err))
		}
	case ChecksumSHA256URL, ChecksumSHA512URL:
		if c.URL == "" {
			errs = append(errs, path+".url is required for type "+c.Type)
		}
	case ChecksumGitHubRelease:
		if !isGitHubRepo(c.Repo) {
			errs = append(errs, path+".repo must be of the form owner/name")
		}
	case ChecksumNone:
	case "":
		errs = append(errs, path+".type is required")
	default:
		errs = append(errs, fmt.Sprintf("%s.type %q is not one of %s, %s, %s, %s, %s",
			path, c.Type, ChecksumHTTPJSON, ChecksumSHA256URL, ChecksumSHA512URL, ChecksumGitHubRelease, ChecksumNone))
	}
	return errs
}

// SupportsSystem reports whether the definition lists system in platforms.
func (d *Definition) SupportsSystem(system string) bool {
	return slices.Contains(d.Platforms, system)
}

// BinDirs returns the directories, relative to the install root, that
// should be added to PATH. It defaults to ["bin"].
func (d *Definition) BinDirs() []string {
	if len(d.Bin) == 0 {
		return []string{"bin"}
	}
	return d.Bin
}

// InstallFor returns the install strategy for version on system. Overrides
// apply in order: the first version override matching version, the
// definition's override for system, and the version override's own
// override for system.
func (d *Definition) InstallFor(system, version string) Install {
	merged := d.Install.merge(&Install{})
	var versionOverride *VersionOverride
	for _, vo := range d.Install.VersionOverrides {
		if vo != nil && MatchesConstraint(version, vo.Versions) {
			versionOverride = vo
			break
		}
	}
	if versionOverride != nil {
		merged = merged.merge(&versionOverride.Install)
	}
	if override := d.Install.Overrides[system]; override != nil {
		merged = merged.merge(override)
	}
	if versionOverride != nil {
		if override := versionOverride.Overrides[system]; override != nil {
			merged = merged.merge(override)
		}
	}
	return merged
}

// merge returns a copy of in with every non-zero field of override applied.
// The result has no overrides of its own. url and lookup are alternatives,
// so an override that sets one clears the other.
func (in *Install) merge(override *Install) Install {
	merged := *in
	merged.Overrides = nil
	merged.VersionOverrides = nil
	if override.Type != "" {
		merged.Type = override.Type
	}
	if override.URL != "" {
		merged.URL = override.URL
		merged.Lookup = nil
	}
	if override.Lookup != nil {
		merged.Lookup = override.Lookup
		merged.URL = ""
		merged.Checksum = nil
	}
	if override.Checksum != nil {
		merged.Checksum = override.Checksum
	}
	if override.StripComponents != nil {
		merged.StripComponents = override.StripComponents
	}
	if override.BinName != "" {
		merged.BinName = override.BinName
	}
	if override.Requires != "" {
		merged.Requires = override.Requires
	}
	if override.Env != nil {
		merged.Env = override.Env
	}
	if override.InstallCmd != "" {
		merged.InstallCmd = override.InstallCmd
	}
	if override.Locate != "" {
		merged.Locate = override.Locate
	}
	if override.Run != "" {
		merged.Run = override.Run
	}
	return merged
}

// ChecksumType returns the effective checksum type, which is "none" when no
// checksum is configured.
func (in *Install) ChecksumType() string {
	if in.Checksum == nil {
		return ChecksumNone
	}
	return in.Checksum.Type
}

// InstallFingerprint identifies everything about the definition that
// affects what an install of version on system contains: the install
// strategy, the bin layout, hooks and platform maps. Activation fields
// (env, shell, create_files) are excluded because they are applied at
// shell time. A changed fingerprint makes Install redo an existing install.
func (d *Definition) InstallFingerprint(system, version string) string {
	install := d.InstallFor(system, version)
	payload, err := json.Marshal(struct {
		Install  Install           `json:"install"`
		Bin      []string          `json:"bin"`
		Binaries []string          `json:"binaries"`
		Hooks    Hooks             `json:"hooks"`
		OSMap    map[string]string `json:"os_map"`
		ArchMap  map[string]string `json:"arch_map"`
	}{install, d.Bin, d.Binaries, d.Hooks, d.OSMap, d.ArchMap})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:8])
}

// UsesScripts reports whether the definition runs shell that the user
// should be told about: script-type sources or strategies, or post-install
// hooks.
func (d *Definition) UsesScripts() bool {
	for _, src := range d.Versions {
		if src.Type == VersionsScript {
			return true
		}
	}
	if d.Install.Type == InstallScript {
		return true
	}
	for _, o := range d.Install.Overrides {
		if o != nil && o.Type == InstallScript {
			return true
		}
	}
	for _, vo := range d.Install.VersionOverrides {
		if vo == nil {
			continue
		}
		if vo.Type == InstallScript {
			return true
		}
		for _, o := range vo.Overrides {
			if o != nil && o.Type == InstallScript {
				return true
			}
		}
	}
	return len(d.Hooks.PostInstall) > 0
}
