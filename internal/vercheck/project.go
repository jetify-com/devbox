// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package vercheck

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"syscall"

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

// autoVersionEnvName records the version that the "auto" policy switched to.
// It distinguishes a DEVBOX_USE_VERSION set by the auto policy from one the
// user set, and stops re-exec loops if the launcher doesn't switch versions.
const autoVersionEnvName = "__DEVBOX_AUTO_VERSION"

// execFunc replaces the current process. We use this variable so that we can
// mock it in tests.
var execFunc = syscall.Exec

// CheckProjectVersion enforces the devbox_version field of the devbox.json at
// configPath against the running devbox version. Depending on the policy, a
// mismatch prints a warning, returns an error, or re-runs the current command
// with the required version (which doesn't return). The DEVBOX_VERSION_POLICY
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
	switch policy {
	case configfile.VersionPolicyError:
		return usererr.New("%s", msg)
	case configfile.VersionPolicyAuto:
		return switchVersion(w, configPath, required, msg)
	default:
		return warnOnce(w, configPath, msg)
	}
}

// switchVersion re-runs the current command with the exact version required
// by devbox_version. It uses the devbox launcher, which downloads the version
// if needed and runs the version named by DEVBOX_USE_VERSION.
func switchVersion(w io.Writer, configPath string, required *configfile.DevboxVersion, msg string) error {
	target, ok := required.ExactVersion()
	if !ok {
		// Only possible with DEVBOX_VERSION_POLICY=auto, since devbox.json
		// validation rejects on_mismatch "auto" with a range.
		return usererr.New(
			"%s\nDevbox can't switch versions automatically because devbox_version is %q, not an exact version like \"0.18.4\".",
			msg, required.Version,
		)
	}

	// Respect a version the user chose explicitly.
	autoVersion := os.Getenv(autoVersionEnvName)
	if useVersion := os.Getenv(envir.DevboxUseVersion); useVersion != "" && useVersion != autoVersion {
		return warnOnce(w, configPath, fmt.Sprintf(
			"%s\nNot switching versions automatically because %s=%s is set.",
			msg, envir.DevboxUseVersion, useVersion,
		))
	}

	if autoVersion == target {
		return usererr.New(
			"%s\nDevbox tried to switch to version %s automatically, but the launcher ran version %s instead.",
			msg, target, currentDevboxVersion,
		)
	}

	launcher := os.Getenv(envir.LauncherPath)
	if launcher == "" {
		return usererr.New(
			"%s\nDevbox can't switch versions automatically because it wasn't started by the devbox launcher. "+
				"Install devbox with `curl -fsSL https://get.jetify.com/devbox | bash` to enable automatic switching.",
			msg,
		)
	}

	slog.Debug("switching devbox version for devbox_version", "from", currentDevboxVersion, "to", target, "launcher", launcher)
	env := withEnv(os.Environ(), map[string]string{
		envir.DevboxUseVersion: target,
		autoVersionEnvName:     target,
	})
	args := append([]string{launcher}, os.Args[1:]...)
	if err := execFunc(launcher, args, env); err != nil {
		return usererr.WithUserMessage(err, "Failed to switch to devbox version %s using the launcher at %s.", target, launcher)
	}
	return nil
}

// warnOnce prints msg as a warning unless it was already printed for the
// project at configPath by this process or a parent devbox process.
func warnOnce(w io.Writer, configPath, msg string) error {
	if os.Getenv(warnedEnvName) == configPath {
		return nil
	}
	ux.Fwarningf(w, "%s\n", msg)
	return os.Setenv(warnedEnvName, configPath)
}

// withEnv returns environ with the variables in vars set, replacing any
// existing values.
func withEnv(environ []string, vars map[string]string) []string {
	result := make([]string, 0, len(environ)+len(vars))
	for _, kv := range environ {
		name, _, _ := strings.Cut(kv, "=")
		if _, ok := vars[name]; !ok {
			result = append(result, kv)
		}
	}
	for name, value := range vars {
		result = append(result, name+"="+value)
	}
	return result
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
