package nix

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/redact"
	"golang.org/x/exp/maps"
)

func StorePathFromHashPart(ctx context.Context, hash, storeAddr string) (string, error) {
	cmd := command("store", "path-from-hash-part", "--store", storeAddr, hash)
	resultBytes, err := cmd.Output(ctx)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(resultBytes)), nil
}

func StorePathsFromInstallable(ctx context.Context, installable string, allowInsecure bool) ([]string, error) {
	defer debug.FunctionTimer().End()
	// --impure for NIXPKGS_ALLOW_UNFREE
	cmd := command("path-info", installable, "--json", "--impure")
	cmd.Env = allowUnfreeEnv(os.Environ())

	if allowInsecure {
		slog.Debug("Setting Allow-insecure env-var\n")
		cmd.Env = allowInsecureEnv(cmd.Env)
	}

	slog.Debug("running cmd", "cmd", cmd)
	resultBytes, err := cmd.Output(ctx)
	if err != nil {
		return nil, err
	}

	paths, err := parseStorePathFromInstallableOutput(resultBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse path-info for %s: %w", installable, err)
	}

	return maps.Keys(paths), nil
}

// StorePathsAreInStore a map of store paths to whether they are in the store.
func StorePathsAreInStore(ctx context.Context, storePaths []string) (map[string]bool, error) {
	defer debug.FunctionTimer().End()
	if len(storePaths) == 0 {
		return map[string]bool{}, nil
	}
	cmd := command("path-info", "--offline", "--json")
	cmd.Args = appendArgs(cmd.Args, storePaths)
	slog.Debug("running cmd", "cmd", cmd)
	output, err := cmd.Output(ctx)
	if err != nil {
		return nil, err
	}

	return parseStorePathFromInstallableOutput(output)
}

// Older nix versions (like 2.17) are an array of objects that contain path and valid fields
type LegacyPathInfo struct {
	Path  string `json:"path"`
	Valid bool   `json:"valid"` // this means path is in store
}

// parseStorePathFromInstallableOutput parses the output of `nix store path-from-installable --json`
// into a map of store paths to whether they are in the store.
// This function is decomposed out of StorePathFromInstallable to make it testable.
func parseStorePathFromInstallableOutput(output []byte) (map[string]bool, error) {
	result := map[string]bool{}

	// Newer nix versions (like 2.20) have output of the form
	// {"<store-path>": {}}
	// Note that values will be null if paths are not in store.
	var modernPathInfo map[string]any
	if err := json.Unmarshal(output, &modernPathInfo); err == nil {
		for path, val := range modernPathInfo {
			result[path] = val != nil
		}
		return result, nil
	}

	var legacyPathInfos []LegacyPathInfo

	if err := json.Unmarshal(output, &legacyPathInfos); err == nil {
		for _, outValue := range legacyPathInfos {
			result[outValue.Path] = outValue.Valid
		}
		return result, nil
	}

	return nil, fmt.Errorf("failed to parse path-info output: %s", output)
}

// DaemonError reports an unsuccessful attempt to connect to the Nix daemon.
type DaemonError struct {
	cmd    string
	stderr []byte
	err    error
}

func (e *DaemonError) Error() string {
	if len(e.stderr) != 0 {
		return e.Redact() + ": " + string(e.stderr)
	}
	return e.Redact()
}

func (e *DaemonError) Unwrap() error {
	return e.err
}

func (e *DaemonError) Redact() string {
	// Don't include e.stderr in redacted messages because it can contain
	// things like paths and usernames.
	if e.cmd != "" {
		return fmt.Sprintf("command %s: %s", e.cmd, e.err)
	}
	return e.err.Error()
}

// DaemonVersion returns the version of the currently running Nix daemon.
func DaemonVersion(ctx context.Context) (string, error) {
	// We only need the version to decide which CLI flags to use. We can
	// ignore the error because an empty version assumes nix.MinVersion.
	cliVersion, _ := Version()

	storeCmd := "ping"
	if cliVersion.AtLeast(Version2_19) {
		// "nix store ping" is deprecated as of 2.19 in favor of
		// "nix store info".
		storeCmd = "info"
	}
	canJSON := cliVersion.AtLeast(Version2_14)

	cmd := command("store", storeCmd, "--store", "daemon")
	if canJSON {
		cmd.Args = append(cmd.Args, "--json")
	}
	out, err := cmd.Output(ctx)

	// ExitError means the command ran, but couldn't connect.
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return "", &DaemonError{
			cmd:    cmd.String(),
			stderr: exitErr.Stderr,
			err:    err,
		}
	}

	// All other errors mean we couldn't launch the Nix CLI (either it is
	// missing or not executable).
	if err != nil {
		return "", redact.Errorf("command %s: %s", redact.Safe(cmd), err)
	}

	if len(out) == 0 {
		return "", redact.Errorf("command %s: empty output", redact.Safe(cmd), err)
	}
	if canJSON {
		info := struct{ Version string }{}
		if err := json.Unmarshal(out, &info); err != nil {
			return "", redact.Errorf("command %s: unmarshal JSON output: %s", redact.Safe(cmd), err)
		}
		return info.Version, nil
	}

	// Example output:
	//
	// Store URL: daemon
	// Version: 2.21.1
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		name, value, found := strings.Cut(line, ": ")
		if found && name == "Version" {
			return value, nil
		}
	}
	return "", redact.Errorf("parse nix daemon version: %s", redact.Safe(lines[0]))
}
