// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"context"
	"fmt"
	"slices"
)

// Artifact is what gets recorded per system in devbox.lock: the exact URL
// to download and its sha256. Sha256 is empty when the definition has no
// checksum source; it is filled in by the first install (trust on first
// use).
type Artifact struct {
	URL    string `json:"url,omitempty"`
	Sha256 string `json:"sha256,omitempty"`
}

// Resolved is the result of resolving a version constraint: the concrete
// version and the artifact for every system the definition supports.
type Resolved struct {
	Version   string
	Artifacts map[string]Artifact
}

// Resolve resolves requested to a concrete version and locks the artifacts
// for every supported system.
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
// every system in the definition's platforms.
func (c *Client) Lock(ctx context.Context, def *Definition, version string) (map[string]Artifact, error) {
	artifacts := make(map[string]Artifact, len(def.Platforms))
	systems := slices.Clone(def.Platforms)
	slices.Sort(systems)
	for _, system := range systems {
		artifact, err := c.lockSystem(ctx, def, version, system)
		if err != nil {
			return nil, fmt.Errorf("%s@%s (%s): %w", def.Name, version, system, err)
		}
		artifacts[system] = artifact
	}
	return artifacts, nil
}

func (c *Client) lockSystem(ctx context.Context, def *Definition, version, system string) (Artifact, error) {
	install := def.InstallFor(system)
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
	rawURL, err := Render(install.URL, vars)
	if err != nil {
		return Artifact{}, err
	}
	vars = vars.withURL(rawURL)

	sha, err := c.fetchChecksum(ctx, install.Checksum, vars)
	if err != nil {
		return Artifact{}, err
	}
	return Artifact{URL: rawURL, Sha256: sha}, nil
}
