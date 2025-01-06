// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/trace"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/redact"
	"go.jetpack.io/devbox/nix/flake"

	"go.jetpack.io/devbox/internal/debug"
)

// ProfilePath contains the contents of the profile generated via `nix-env --profile ProfilePath <command>`
// or `nix profile install --profile ProfilePath <package...>`
// Instead of using directory, prefer using the devbox.ProfileDir() function that ensures the directory exists.
const ProfilePath = ".devbox/nix/profile/default"

type PrintDevEnvOut struct {
	Variables map[string]Variable // the key is the name.
}

type Variable struct {
	Type  string // valid types are var, exported, and array.
	Value any    // can be a string or an array of strings (iff type is array).
}

type PrintDevEnvArgs struct {
	FlakeDir             string
	PrintDevEnvCachePath string
	UsePrintDevEnvCache  bool
}

// PrintDevEnv calls `nix print-dev-env -f <path>` and returns its output. The output contains
// all the environment variables and bash functions required to create a nix shell.
func (*NixInstance) PrintDevEnv(ctx context.Context, args *PrintDevEnvArgs) (*PrintDevEnvOut, error) {
	defer debug.FunctionTimer().End()
	defer trace.StartRegion(ctx, "nixPrintDevEnv").End()

	var data []byte
	var err error
	var out PrintDevEnvOut

	if args.UsePrintDevEnvCache {
		data, err = os.ReadFile(args.PrintDevEnvCachePath)
		if err == nil {
			if err := json.Unmarshal(data, &out); err != nil {
				return nil, errors.WithStack(err)
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, errors.WithStack(err)
		}
	}

	flakeDirResolved, err := filepath.EvalSymlinks(args.FlakeDir)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	ref := flake.Ref{Type: flake.TypePath, Path: flakeDirResolved}

	if len(data) == 0 {
		cmd := Command("print-dev-env", "--json")
		if featureflag.ImpurePrintDevEnv.Enabled() {
			cmd.Args = append(cmd.Args, "--impure")
		}
		cmd.Args = append(cmd.Args, ref)
		slog.Debug("running print-dev-env cmd", "cmd", cmd)
		data, err = cmd.Output(ctx)
		if insecure, insecureErr := IsExitErrorInsecurePackage(err, "" /*pkgName*/, "" /*installable*/); insecure {
			return nil, insecureErr
		} else if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(data, &out); err != nil {
			return nil, redact.Errorf("unmarshal nix print-dev-env output: %w", redact.Safe(err))
		}

		if err = savePrintDevEnvCache(args.PrintDevEnvCachePath, out); err != nil {
			return nil, redact.Errorf("savePrintDevEnvCache: %w", redact.Safe(err))
		}
	}

	return &out, nil
}

func savePrintDevEnvCache(path string, out PrintDevEnvOut) error {
	data, err := json.Marshal(out)
	if err != nil {
		return errors.WithStack(err)
	}

	_ = os.WriteFile(path, data, 0o644)
	return nil
}

// FlakeNixpkgs returns a flakes-compatible reference to the nixpkgs registry.
// TODO savil. Ensure this works with the nixed cache service.
func FlakeNixpkgs(commit string) string {
	// Using nixpkgs/<commit> means:
	// The nixpkgs entry in the flake registry, with its Git revision overridden to a specific value.
	return "nixpkgs/" + commit
}

func ExperimentalFlags() []string {
	options := []string{"nix-command", "flakes", "fetch-closure"}
	return []string{
		"--extra-experimental-features", "ca-derivations",
		"--option", "experimental-features", strings.Join(options, " "),
	}
}

func SystemIsLinux() bool {
	return strings.Contains(System(), "linux")
}

var nixPlatforms = []string{
	"aarch64-darwin",
	"aarch64-linux",
	"i686-linux",
	"x86_64-darwin",
	"x86_64-linux",
	// not technically supported, but should work?
	// ref. https://nixos.wiki/wiki/Nix_on_ARM
	// ref. https://github.com/jetify-com/devbox/pull/1300
	"armv7l-linux",
}

// EnsureValidPlatform returns an error if the platform is not supported by nix.
// https://nixos.org/manual/nix/stable/installation/supported-platforms.html
func EnsureValidPlatform(platforms ...string) error {
	ensureValid := func(platform string) error {
		for _, p := range nixPlatforms {
			if p == platform {
				return nil
			}
		}
		return usererr.New("Unsupported platform: %s. Valid platforms are: %v", platform, nixPlatforms)
	}

	for _, p := range platforms {
		if err := ensureValid(p); err != nil {
			return err
		}
	}
	return nil
}

// Warning: be careful using the bins in default/bin, they won't always match bins
// produced by the flakes.nix. Use devbox.NixBins() instead.
func ProfileBinPath(projectDir string) string {
	return filepath.Join(projectDir, ProfilePath, "bin")
}

func IsExitErrorInsecurePackage(err error, pkgNameOrEmpty, installableOrEmpty string) (bool, error) {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		if strings.Contains(string(exitErr.Stderr), "is marked as insecure") {
			packageRegex := regexp.MustCompile(`Package ([^ ]+)`)
			packageMatch := packageRegex.FindStringSubmatch(string(exitErr.Stderr))

			knownVulnerabilities := []string{}
			if installableOrEmpty != "" {
				knownVulnerabilities = PackageKnownVulnerabilities(installableOrEmpty)
			}

			insecurePackages := parseInsecurePackagesFromExitError(string(exitErr.Stderr))

			// Construct the error message.
			errMessages := []string{}
			errMessages = append(errMessages, fmt.Sprintf("Package %s is insecure.", packageMatch[1]))
			if len(knownVulnerabilities) > 0 {
				errMessages = append(errMessages,
					fmt.Sprintf("Known vulnerabilities:\n%s", strings.Join(knownVulnerabilities, "\n")))
			}
			pkgName := pkgNameOrEmpty
			if pkgName == "" {
				pkgName = "<pkg>"
			}
			errMessages = append(errMessages,
				fmt.Sprintf("To override, use `devbox add %s --allow-insecure=%s`", pkgName, strings.Join(insecurePackages, ", ")))

			return true, usererr.New("%s", strings.Join(errMessages, "\n\n"))
		}
	}
	return false, nil
}

func parseInsecurePackagesFromExitError(errorMsg string) []string {
	insecurePackages := []string{}

	// permittedRegex is designed to match the following:
	// permittedInsecurePackages = [
	//    "package-one"
	//    "package-two"
	// ];
	permittedRegex := regexp.MustCompile(`permittedInsecurePackages\s*=\s*\[([\s\S]*?)\]`)
	permittedMatch := permittedRegex.FindStringSubmatch(errorMsg)
	if len(permittedMatch) > 1 {
		packagesList := permittedMatch[1]
		// pick out the package name strings inside the quotes
		packageMatches := regexp.MustCompile(`"([^"]+)"`).FindAllStringSubmatch(packagesList, -1)

		// Extract the insecure package names from the matches
		for _, packageMatch := range packageMatches {
			if len(packageMatch) > 1 {
				insecurePackages = append(insecurePackages, packageMatch[1])
			}
		}
	}

	return insecurePackages
}

var ErrUnknownServiceManager = errors.New("unknown service manager")

func restartDaemon(ctx context.Context) error {
	if runtime.GOOS != "darwin" {
		err := fmt.Errorf("don't know how to restart nix daemon: %w", ErrUnknownServiceManager)
		return &DaemonError{err: err}
	}

	cmd := exec.CommandContext(ctx, "launchctl", "bootout", "system", "/Library/LaunchDaemons/org.nixos.nix-daemon.plist")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &DaemonError{
			cmd:    cmd.String(),
			stderr: out,
			err:    fmt.Errorf("stop nix daemon: %w", err),
		}
	}
	cmd = exec.CommandContext(ctx, "launchctl", "bootstrap", "system", "/Library/LaunchDaemons/org.nixos.nix-daemon.plist")
	out, err = cmd.CombinedOutput()
	if err != nil {
		return &DaemonError{
			cmd:    cmd.String(),
			stderr: out,
			err:    fmt.Errorf("start nix daemon: %w", err),
		}
	}

	// TODO(gcurtis): poll for daemon to come back instead.
	time.Sleep(2 * time.Second)
	return nil
}

// FixInstallableArgs removes the narHash and lastModifiedDate query parameters
// from any args that are valid installables and the Nix version is <2.25.
// Otherwise it returns them unchanged.
//
// This fixes an issues with some older versions of Nix where specifying a
// narHash without a lastModifiedDate results in an error.
func FixInstallableArgs(args []string) {
	if AtLeast(Version2_25) {
		return
	}

	for i := range args {
		parsed, _ := flake.ParseInstallable(args[i])
		if parsed.Ref.NARHash == "" && parsed.Ref.LastModified == 0 {
			continue
		}
		if parsed.Ref.NARHash != "" && parsed.Ref.LastModified != 0 {
			continue
		}

		parsed.Ref.NARHash = ""
		parsed.Ref.LastModified = 0
		args[i] = parsed.String()
	}
}

// fixInstallableArg calls fixInstallableArgs with a single argument.
func FixInstallableArg(arg string) string {
	args := []string{arg}
	FixInstallableArgs(args)
	return args[0]
}
