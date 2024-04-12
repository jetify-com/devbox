package nix

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"os/user"
	"slices"
	"strings"

	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/redact"
)

// Config is a parsed Nix configuration.
type Config struct {
	ExperimentalFeatures ConfigField[[]string] `json:"experimental-features"`
	Substitute           ConfigField[bool]     `json:"substitute"`
	Substituters         ConfigField[[]string] `json:"substituters"`
	System               ConfigField[string]   `json:"system"`
	TrustedSubstituters  ConfigField[[]string] `json:"trusted-substituters"`
	TrustedUsers         ConfigField[[]string] `json:"trusted-users"`
}

// ConfigField is a Nix configuration setting.
type ConfigField[T any] struct {
	Value T `json:"value"`
}

// CurrentConfig reads the current Nix configuration.
func CurrentConfig(ctx context.Context) (Config, error) {
	// `nix show-config` is deprecated in favor of `nix config show`, but we
	// want to remain compatible with older Nix versions.
	cmd := commandContext(ctx, "show-config", "--json")
	out, err := cmd.Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && len(exitErr.Stderr) != 0 {
		return Config{}, redact.Errorf("command %s: %v: %s", redact.Safe(cmd), err, exitErr.Stderr)
	}
	if err != nil {
		return Config{}, redact.Errorf("command %s: %v", cmd, err)
	}
	cfg := Config{}
	if err := json.Unmarshal(out, &cfg); err != nil {
		return Config{}, redact.Errorf("unmarshal JSON output from %s: %v", redact.Safe(cmd), err)
	}
	return cfg, nil
}

// IsUserTrusted reports if the current OS user is in the trusted-users list. If
// there are any groups in the list, it also checks if the user belongs to any
// of them.
func (c Config) IsUserTrusted(ctx context.Context) (bool, error) {
	trusted := c.TrustedUsers.Value
	if len(trusted) == 0 {
		return false, nil
	}

	current, err := user.Current()
	if err != nil {
		return false, redact.Errorf("lookup current user: %v", err)
	}
	if slices.Contains(trusted, current.Username) {
		return true, nil
	}

	// trusted-user entries that start with an @ are group names
	// (for example, @wheel). Lookup each group ID to see if the user
	// belongs to a trusted group.
	var currentGids []string
	for i := range trusted {
		groupName := strings.TrimPrefix(trusted[i], "@")
		if groupName == trusted[i] || groupName == "" {
			continue
		}

		group, err := user.LookupGroup(groupName)
		var unknownErr user.UnknownGroupError
		if errors.As(err, &unknownErr) {
			debug.Log("skipping unknown trusted-user group %q found in nix.conf", groupName)
			continue
		}
		if err != nil {
			return false, redact.Errorf("lookup trusted-user group from nix.conf: %v", err)
		}

		// Be lazy about looking up the current user's groups until we
		// encounter one in the trusted-users list.
		if currentGids == nil {
			currentGids, err = current.GroupIds()
			if err != nil {
				return false, redact.Errorf("lookup current user group IDs: %v", err)
			}
		}
		if slices.Contains(currentGids, group.Gid) {
			return true, nil
		}
	}
	return false, nil
}
