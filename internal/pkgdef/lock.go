// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// ErrNoArtifact is returned when the vendor publishes no artifact of a
// version for a system, for example an old release that predates a
// platform.
var ErrNoArtifact = errors.New("no artifact for this system")

// Artifact is what gets recorded per system in devbox.lock: the exact URL
// to download and its digest. Vendors publish sha256 or sha512 digests, so
// at most one of them is usually set. Both are empty when the definition
// has no checksum source; the first install fills in Sha256 (trust on
// first use).
type Artifact struct {
	URL    string `json:"url,omitempty"`
	Sha256 string `json:"sha256,omitempty"`
	Sha512 string `json:"sha512,omitempty"`
}

// HasDigest reports whether the artifact has a digest to verify against.
func (a Artifact) HasDigest() bool {
	return a.Sha256 != "" || a.Sha512 != ""
}

// Resolved is the result of resolving a version constraint: the concrete
// version and the artifact for every supported system the vendor publishes
// it for.
type Resolved struct {
	Version   string
	Artifacts map[string]Artifact
}

// Resolve resolves requested to a concrete version and locks its artifacts.
func (c *Client) Resolve(ctx context.Context, def *Definition, requested string) (*Resolved, error) {
	version, err := c.ResolveVersion(ctx, def, requested)
	if err != nil {
		return nil, err
	}
	artifacts, err := c.Lock(ctx, def, version)
	if err != nil {
		return nil, err
	}
	return &Resolved{Version: version, Artifacts: artifacts}, nil
}

// Lock renders the artifact URL and fetches the checksum of version for
// every system in the definition's platforms. Systems the vendor publishes
// no artifact for are left out; it is an error only when no system has
// one.
func (c *Client) Lock(ctx context.Context, def *Definition, version string) (map[string]Artifact, error) {
	artifacts := make(map[string]Artifact, len(def.Platforms))
	systems := slices.Clone(def.Platforms)
	slices.Sort(systems)
	var missing []string
	for _, system := range systems {
		artifact, err := c.lockSystem(ctx, def, version, system)
		if errors.Is(err, ErrNoArtifact) {
			missing = append(missing, fmt.Sprintf("%s: %v", system, err))
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("%s@%s (%s): %w", def.Name, version, system, err)
		}
		artifacts[system] = artifact
	}
	if len(artifacts) == 0 {
		return nil, fmt.Errorf("%s@%s: %w on any supported system:\n  %s",
			def.Name, version, ErrNoArtifact, strings.Join(missing, "\n  "))
	}
	return artifacts, nil
}

func (c *Client) lockSystem(ctx context.Context, def *Definition, version, system string) (Artifact, error) {
	install := def.InstallFor(system, version)
	switch install.Type {
	case InstallArchive, InstallBinary:
	case InstallManager, InstallScript:
		return Artifact{}, fmt.Errorf("install.type %q: %w", install.Type, ErrNotImplemented)
	default:
		return Artifact{}, fmt.Errorf("install.type %q is not supported", install.Type)
	}

	vars, err := def.TemplateVars(system, version)
	if err != nil {
		return Artifact{}, err
	}
	if install.Lookup != nil {
		return c.lookupArtifact(ctx, def, install.Lookup, vars)
	}

	// The URL is recorded as rendered (a relative file:// URL stays
	// relative) so that devbox.lock is portable; it is resolved against the
	// definition's directory when fetching.
	rawURL, err := Render(install.URL, vars)
	if err != nil {
		return Artifact{}, err
	}
	vars = vars.withURL(rawURL)

	d, err := c.fetchChecksum(ctx, def, install.Checksum, vars)
	if err != nil {
		return Artifact{}, err
	}
	artifact := Artifact{URL: rawURL}
	d.apply(&artifact)
	return artifact, nil
}

// lookupArtifact evaluates an install lookup: it renders the index URL and
// jq expression for version and system and reads "url" and "checksum" from
// the first object the expression produces. An index that does not exist
// or has no entry means the vendor has no artifact for this system.
func (c *Client) lookupArtifact(ctx context.Context, def *Definition, lookup *Lookup, vars Vars) (Artifact, error) {
	rawURL, err := Render(lookup.URL, vars)
	if err != nil {
		return Artifact{}, err
	}
	rawURL = def.ResolveURL(rawURL)
	expr, err := Render(lookup.Select, vars)
	if err != nil {
		return Artifact{}, err
	}
	results, err := c.queryJSON(ctx, rawURL, expr)
	if errors.Is(err, ErrNotFound) {
		return Artifact{}, fmt.Errorf("lookup: %w: %w", ErrNoArtifact, err)
	}
	if err != nil {
		return Artifact{}, fmt.Errorf("lookup: %w", err)
	}
	if len(results) == 0 {
		return Artifact{}, fmt.Errorf("lookup: %w: %s has no entry for %s on %s", ErrNoArtifact, rawURL, vars.Version, vars.System)
	}
	obj, ok := results[0].(map[string]any)
	if !ok {
		return Artifact{}, fmt.Errorf("lookup: %s: jq expression %q must produce objects with \"url\" and \"checksum\", got %T",
			rawURL, expr, results[0])
	}
	artifactURL, _ := obj["url"].(string)
	if artifactURL == "" {
		return Artifact{}, fmt.Errorf("lookup: %s: jq expression %q produced an object without \"url\"", rawURL, expr)
	}
	artifact := Artifact{URL: artifactURL}
	if checksum, _ := obj["checksum"].(string); checksum != "" {
		d, err := parseDigest(checksum, rawURL, "")
		if err != nil {
			return Artifact{}, err
		}
		d.apply(&artifact)
	}
	return artifact, nil
}
