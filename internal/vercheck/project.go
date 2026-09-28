// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package vercheck

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"

	"go.jetify.com/devbox/internal/boxcli/usererr"
	"go.jetify.com/devbox/internal/devconfig/configfile"
	"go.jetify.com/devbox/internal/envir"
	"go.jetify.com/devbox/internal/ux"
)

// warnedEnvName records the config path of the last project whose
// devbox_version mismatch was reported, so that nested devbox commands don't
// print the same warning again.
const warnedEnvName = "__DEVBOX_VERSION_MISMATCH_WARNED"

// CheckProjectVersion enforces the devbox_version field of the devbox.json at
// configPath against the running devbox version. Depending on the policy, a
// mismatch prints a warning or returns an error. The DEVBOX_VERSION_POLICY
// environment variable overrides the policy in devbox.json.
func CheckProjectVersion(w io.Writer, configPath string, required *configfile.DevboxVersion) error {
	if required == nil || isDevBuild {
		return nil
	}

	policy := required.Policy()
	if env := os.Getenv(envir.DevboxVersionPolicy); env != "" {
		p, err := configfile.ParseVersionPolicy(env)
		if err != nil {
			return usererr.New("Invalid %s: %v", envir.DevboxVersionPolicy, err)
		}
		policy = p
	}
	if policy == configfile.VersionPolicyOff {
		return nil
	}

	constraint, err := required.Constraint()
	if err != nil {
		return err
	}
	current, err := semver.NewVersion(currentDevboxVersion)
	if err != nil {
		slog.Debug("skipping devbox_version check: can't parse running version", "version", currentDevboxVersion, "err", err)
		return nil
	}
	if constraint.Check(current) {
		return nil
	}

	msg := mismatchMessage(configPath, required, constraint)
	if policy == configfile.VersionPolicyError {
		return usererr.New("%s", msg)
	}
	if os.Getenv(warnedEnvName) == configPath {
		return nil
	}
	ux.Fwarningf(w, "%s\n", msg)
	return os.Setenv(warnedEnvName, configPath)
}

func mismatchMessage(configPath string, required *configfile.DevboxVersion, constraint *semver.Constraints) string {
	var msg strings.Builder
	fmt.Fprintf(
		&msg,
		"This project requires devbox %s (devbox_version in %s), but you are running %s.\n",
		required.Version, configPath, currentDevboxVersion,
	)
	if exact, ok := required.ExactVersion(); ok {
		fmt.Fprintf(&msg, "Set %s=%s to run the required version.", envir.DevboxUseVersion, exact)
	} else if latestSatisfies(constraint) {
		fmt.Fprintf(
			&msg,
			"Run `devbox version update`, or set %s to a version that satisfies %q.",
			envir.DevboxUseVersion, required.Version,
		)
	} else {
		fmt.Fprintf(&msg, "Set %s to a version that satisfies %q.", envir.DevboxUseVersion, required.Version)
	}
	fmt.Fprintf(&msg, " To skip this check, set %s=off.", envir.DevboxVersionPolicy)
	return msg.String()
}

// latestSatisfies reports whether the latest devbox release satisfies
// constraint, meaning `devbox version update` would fix a mismatch. It's false
// when the latest version is unknown or when the constraint excludes it (for
// example, an upper bound that the running version is already past).
func latestSatisfies(constraint *semver.Constraints) bool {
	latest, err := semver.NewVersion(latestVersion())
	return err == nil && constraint.Check(latest)
}
