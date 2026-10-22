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
// e.g. "pkg:./go.pkg.json@1.22".
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
	ChecksumHTTPJSON  = "http_json"
	ChecksumSHA256URL = "sha256_url"
	ChecksumNone      = "none"
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

	Versions VersionSource `json:"versions"`
	Install  Install       `json:"install"`

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

// VersionSource describes how to list the available versions of a package.
type VersionSource struct {
	Type string `json:"type"`

	// static
	Versions []string `json:"versions,omitempty"`

	// http_json
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

	// archive and binary
	URL      string    `json:"url,omitempty"`
	Checksum *Checksum `json:"checksum,omitempty"`

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
}

// Checksum describes where to find the sha256 of an artifact.
type Checksum struct {
	Type   string `json:"type"`
	URL    string `json:"url,omitempty"`
	Select string `json:"select,omitempty"`
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

	errs = append(errs, d.Versions.validate()...)
	errs = append(errs, d.Install.validate("install")...)
	for sys, override := range d.Install.Overrides {
		if !slices.Contains(d.Platforms, sys) {
			fail("install.overrides: %q is not in platforms", sys)
		}
		if override == nil {
			fail("install.overrides.%s must be an object", sys)
			continue
		}
		if len(override.Overrides) > 0 {
			fail("install.overrides.%s may not contain overrides", sys)
		}
		merged := d.Install.merge(override)
		errs = append(errs, merged.validate("install.overrides."+sys)...)
	}

	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("invalid package definition %q:\n  - %s", d.Name, strings.Join(errs, "\n  - "))
}

func (v *VersionSource) validate() []string {
	var errs []string
	switch v.Type {
	case VersionsStatic:
		if len(v.Versions) == 0 {
			errs = append(errs, "versions.versions must list at least one version")
		}
	case VersionsHTTPJSON:
		if v.URL == "" {
			errs = append(errs, "versions.url is required for type http_json")
		}
		if v.Select == "" {
			errs = append(errs, "versions.select is required for type http_json")
		} else if _, err := compileJQ(v.Select); err != nil {
			errs = append(errs, fmt.Sprintf("versions.select: %v", err))
		}
	case VersionsGitHubReleases:
		if _, repo, _ := strings.Cut(v.Repo, "/"); !githubRepoRegexp.MatchString(v.Repo) || repo == "." || repo == ".." {
			errs = append(errs, "versions.repo must be of the form owner/name")
		}
	case VersionsScript:
		if v.Run == "" {
			errs = append(errs, "versions.run is required for type script")
		}
	case "":
		errs = append(errs, "versions.type is required")
	default:
		errs = append(errs, fmt.Sprintf("versions.type %q is not one of %s, %s, %s, %s",
			v.Type, VersionsStatic, VersionsHTTPJSON, VersionsGitHubReleases, VersionsScript))
	}
	return errs
}

func (in *Install) validate(path string) []string {
	var errs []string
	switch in.Type {
	case InstallArchive, InstallBinary:
		if in.URL == "" {
			errs = append(errs, path+".url is required for type "+in.Type)
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
		} else if _, err := compileJQ(c.Select); err != nil {
			errs = append(errs, fmt.Sprintf("%s.select: %v", path, err))
		}
	case ChecksumSHA256URL:
		if c.URL == "" {
			errs = append(errs, path+".url is required for type sha256_url")
		}
	case ChecksumNone:
	case "":
		errs = append(errs, path+".type is required")
	default:
		errs = append(errs, fmt.Sprintf("%s.type %q is not one of %s, %s, %s",
			path, c.Type, ChecksumHTTPJSON, ChecksumSHA256URL, ChecksumNone))
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

// InstallFor returns the install strategy for system with any override for
// that system applied.
func (d *Definition) InstallFor(system string) Install {
	override := d.Install.Overrides[system]
	if override == nil {
		merged := d.Install
		merged.Overrides = nil
		return merged
	}
	return d.Install.merge(override)
}

// merge returns a copy of in with every non-zero field of override applied.
func (in *Install) merge(override *Install) Install {
	merged := *in
	merged.Overrides = nil
	if override.Type != "" {
		merged.Type = override.Type
	}
	if override.URL != "" {
		merged.URL = override.URL
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
// affects what an install of a version on system contains: the install
// strategy, the bin layout, hooks and platform maps. Activation fields
// (env, shell, create_files) are excluded because they are applied at
// shell time. A changed fingerprint makes Install redo an existing install.
func (d *Definition) InstallFingerprint(system string) string {
	install := d.InstallFor(system)
	payload, err := json.Marshal(struct {
		Install Install           `json:"install"`
		Bin     []string          `json:"bin"`
		Hooks   Hooks             `json:"hooks"`
		OSMap   map[string]string `json:"os_map"`
		ArchMap map[string]string `json:"arch_map"`
	}{install, d.Bin, d.Hooks, d.OSMap, d.ArchMap})
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
	if d.Versions.Type == VersionsScript || d.Install.Type == InstallScript {
		return true
	}
	for _, o := range d.Install.Overrides {
		if o != nil && o.Type == InstallScript {
			return true
		}
	}
	return len(d.Hooks.PostInstall) > 0
}
