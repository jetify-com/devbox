// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package configfile

import (
	"encoding/json"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"
	"go.jetify.com/devbox/internal/boxcli/usererr"
)

// VersionPolicy controls what devbox does when the running devbox version
// doesn't satisfy a project's devbox_version constraint.
type VersionPolicy string

const (
	// VersionPolicyOff disables the check. It can only be set with the
	// DEVBOX_VERSION_POLICY environment variable, not in devbox.json.
	VersionPolicyOff VersionPolicy = "off"
	// VersionPolicyWarn prints a warning and continues.
	VersionPolicyWarn VersionPolicy = "warn"
	// VersionPolicyError fails the command.
	VersionPolicyError VersionPolicy = "error"
)

// ConfigVersionPolicies are the values allowed for devbox_version.on_mismatch.
var ConfigVersionPolicies = []VersionPolicy{
	VersionPolicyWarn,
	VersionPolicyError,
}

// DevboxVersion is the devbox_version field of devbox.json. It is either a
// version constraint string, which uses the default "warn" policy:
//
//	"devbox_version": "^0.18.0"
//
// or an object with an explicit policy:
//
//	"devbox_version": {"version": "0.18.4", "on_mismatch": "error"}
type DevboxVersion struct {
	// Version is a semver constraint (e.g. "0.18.4", "^0.18.0",
	// ">=0.17.0 <0.19.0") that the running devbox version must satisfy.
	Version string `json:"version"`

	// OnMismatch is the policy to apply when the running devbox version
	// doesn't satisfy Version. Defaults to "warn".
	OnMismatch VersionPolicy `json:"on_mismatch,omitempty"`

	// isShorthand records whether the field was written as a plain string so
	// that marshaling preserves the original form.
	isShorthand bool
}

// Policy returns the configured on_mismatch policy, or the default if unset.
func (d *DevboxVersion) Policy() VersionPolicy {
	if d.OnMismatch == "" {
		return VersionPolicyWarn
	}
	return d.OnMismatch
}

// Constraint parses Version as a semver constraint.
func (d *DevboxVersion) Constraint() (*semver.Constraints, error) {
	c, err := semver.NewConstraint(d.Version)
	if err != nil {
		return nil, usererr.New(
			"Invalid devbox_version %q in devbox.json: %v. Use a version like \"0.18.4\" or a constraint like \"^0.18.0\".",
			d.Version, err,
		)
	}
	return c, nil
}

// ExactVersion returns Version as an exact version (without a leading "v")
// and true if Version is a single exact version rather than a range.
func (d *DevboxVersion) ExactVersion() (string, bool) {
	v := strings.TrimPrefix(strings.TrimSpace(d.Version), "v")
	if _, err := semver.StrictNewVersion(v); err != nil {
		return "", false
	}
	return v, true
}

func (d *DevboxVersion) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		d.isShorthand = true
		return json.Unmarshal(data, &d.Version)
	}
	type devboxVersion DevboxVersion
	return json.Unmarshal(data, (*devboxVersion)(d))
}

func (d DevboxVersion) MarshalJSON() ([]byte, error) {
	if d.isShorthand {
		return json.Marshal(d.Version)
	}
	type devboxVersion DevboxVersion
	return json.Marshal(devboxVersion(d))
}

func validateDevboxVersion(cfg *ConfigFile) error {
	required := cfg.DevboxVersion
	if required == nil {
		return nil
	}
	if strings.TrimSpace(required.Version) == "" {
		return usererr.New("devbox_version in devbox.json must specify a version")
	}
	if _, err := required.Constraint(); err != nil {
		return err
	}
	if required.OnMismatch != "" && !slices.Contains(ConfigVersionPolicies, required.OnMismatch) {
		return usererr.New(
			"Invalid devbox_version.on_mismatch %q in devbox.json. Valid values are %q and %q.",
			required.OnMismatch, VersionPolicyWarn, VersionPolicyError,
		)
	}
	return nil
}

// ParseVersionPolicy parses a policy from the DEVBOX_VERSION_POLICY
// environment variable. Unlike devbox.json, it also accepts "off".
func ParseVersionPolicy(s string) (VersionPolicy, error) {
	p := VersionPolicy(strings.ToLower(strings.TrimSpace(s)))
	if p == VersionPolicyOff || slices.Contains(ConfigVersionPolicies, p) {
		return p, nil
	}
	return "", errors.Errorf(
		"invalid policy %q: valid values are %q, %q, and %q",
		s, VersionPolicyOff, VersionPolicyWarn, VersionPolicyError,
	)
}
