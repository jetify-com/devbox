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
	AcceptFlakeConfig                     ConfigField[bool]              `json:"accept-flake-config"`
	AccessTokens                          ConfigField[map[string]string] `json:"access-tokens"`
	AllowDirty                            ConfigField[bool]              `json:"allow-dirty"`
	AllowImportFromDerivation             ConfigField[bool]              `json:"allow-import-from-derivation"`
	AllowSymlinkedStore                   ConfigField[bool]              `json:"allow-symlinked-store"`
	AllowUnsafeNativeCodeDuringEvaluation ConfigField[bool]              `json:"allow-unsafe-native-code-during-evaluation"`
	AllowedImpureHostDeps                 ConfigField[[]string]          `json:"allowed-impure-host-deps"`
	AllowedURIs                           ConfigField[[]string]          `json:"allowed-uris"`
	AllowedUsers                          ConfigField[[]string]          `json:"allowed-users"`
	AlwaysAllowSubstitutes                ConfigField[bool]              `json:"always-allow-substitutes"`
	AutoAllocateUIDs                      ConfigField[bool]              `json:"auto-allocate-uids"`
	AutoOptimiseStore                     ConfigField[bool]              `json:"auto-optimise-store"`
	BashPrompt                            ConfigField[string]            `json:"bash-prompt"`
	BashPromptPrefix                      ConfigField[string]            `json:"bash-prompt-prefix"`
	BashPromptSuffix                      ConfigField[string]            `json:"bash-prompt-suffix"`
	BuildHook                             ConfigField[[]string]          `json:"build-hook"`
	BuildPollInterval                     ConfigField[int]               `json:"build-poll-interval"`
	BuildUsersGroup                       ConfigField[string]            `json:"build-users-group"`
	Builders                              ConfigField[string]            `json:"builders"`
	BuildersUseSubstitutes                ConfigField[bool]              `json:"builders-use-substitutes"`
	CommitLockfileSummary                 ConfigField[string]            `json:"commit-lockfile-summary"`
	CompressBuildLog                      ConfigField[bool]              `json:"compress-build-log"`
	ConnectTimeout                        ConfigField[int]               `json:"connect-timeout"`
	Cores                                 ConfigField[int]               `json:"cores"`
	DarwinLogSandboxViolations            ConfigField[bool]              `json:"darwin-log-sandbox-violations"`
	DebuggerOnTrace                       ConfigField[bool]              `json:"debugger-on-trace"`
	DiffHook                              ConfigField[string]            `json:"diff-hook"`
	DownloadAttempts                      ConfigField[int]               `json:"download-attempts"`
	DownloadSpeed                         ConfigField[int]               `json:"download-speed"`
	EvalCache                             ConfigField[bool]              `json:"eval-cache"`
	EvalSystem                            ConfigField[string]            `json:"eval-system"`
	ExperimentalFeatures                  ConfigField[[]string]          `json:"experimental-features"`
	ExtraPlatforms                        ConfigField[[]string]          `json:"extra-platforms"`
	Fallback                              ConfigField[bool]              `json:"fallback"`
	FlakeRegistry                         ConfigField[string]            `json:"flake-registry"`
	FsyncMetadata                         ConfigField[bool]              `json:"fsync-metadata"`
	GCReservedSpace                       ConfigField[int]               `json:"gc-reserved-space"`
	HashedMirrors                         ConfigField[[]string]          `json:"hashed-mirrors"`
	HTTPConnections                       ConfigField[int]               `json:"http-connections"`
	HTTP2                                 ConfigField[bool]              `json:"http2"`
	IDCount                               ConfigField[int]               `json:"id-count"`
	IgnoreTry                             ConfigField[bool]              `json:"ignore-try"`
	ImpersonateLinux26                    ConfigField[bool]              `json:"impersonate-linux-26"`
	ImpureEnv                             ConfigField[map[string]string] `json:"impure-env"`
	KeepBuildLog                          ConfigField[bool]              `json:"keep-build-log"`
	KeepDerivations                       ConfigField[bool]              `json:"keep-derivations"`
	KeepEnvDerivations                    ConfigField[bool]              `json:"keep-env-derivations"`
	KeepFailed                            ConfigField[bool]              `json:"keep-failed"`
	KeepGoing                             ConfigField[bool]              `json:"keep-going"`
	KeepOutputs                           ConfigField[bool]              `json:"keep-outputs"`
	LogLines                              ConfigField[int]               `json:"log-lines"`
	MaxBuildLogSize                       ConfigField[int]               `json:"max-build-log-size"`
	MaxCallDepth                          ConfigField[int]               `json:"max-call-depth"`
	MaxFree                               ConfigField[uint64]            `json:"max-free"`
	MaxJobs                               ConfigField[int]               `json:"max-jobs"`
	MaxSilentTime                         ConfigField[int]               `json:"max-silent-time"`
	MaxSubstitutionJobs                   ConfigField[int]               `json:"max-substitution-jobs"`
	MinFree                               ConfigField[int]               `json:"min-free"`
	MinFreeCheckInterval                  ConfigField[int]               `json:"min-free-check-interval"`
	NarBufferSize                         ConfigField[int]               `json:"nar-buffer-size"`
	NarinfoCacheNegativeTTL               ConfigField[int]               `json:"narinfo-cache-negative-ttl"`
	NarinfoCachePositiveTTL               ConfigField[int]               `json:"narinfo-cache-positive-ttl"`
	NetrcFile                             ConfigField[string]            `json:"netrc-file"`
	NixPath                               ConfigField[[]string]          `json:"nix-path"`
	PluginFiles                           ConfigField[[]string]          `json:"plugin-files"`
	PostBuildHook                         ConfigField[string]            `json:"post-build-hook"`
	PreBuildHook                          ConfigField[string]            `json:"pre-build-hook"`
	PreallocateContents                   ConfigField[bool]              `json:"preallocate-contents"`
	PrintMissing                          ConfigField[bool]              `json:"print-missing"`
	PureEval                              ConfigField[bool]              `json:"pure-eval"`
	RequireDropSupplementaryGroups        ConfigField[bool]              `json:"require-drop-supplementary-groups"`
	RequireSigs                           ConfigField[bool]              `json:"require-sigs"`
	RestrictEval                          ConfigField[bool]              `json:"restrict-eval"`
	RunDiffHook                           ConfigField[bool]              `json:"run-diff-hook"`
	Sandbox                               ConfigField[bool]              `json:"sandbox"`
	SandboxFallback                       ConfigField[bool]              `json:"sandbox-fallback"`
	SandboxPaths                          ConfigField[[]string]          `json:"sandbox-paths"`
	SecretKeyFiles                        ConfigField[[]string]          `json:"secret-key-files"`
	ShowTrace                             ConfigField[bool]              `json:"show-trace"`
	SSLCertFile                           ConfigField[string]            `json:"ssl-cert-file"`
	StalledDownloadTimeout                ConfigField[int]               `json:"stalled-download-timeout"`
	StartID                               ConfigField[int]               `json:"start-id"`
	Store                                 ConfigField[string]            `json:"store"`
	Substitute                            ConfigField[bool]              `json:"substitute"`
	Substituters                          ConfigField[[]string]          `json:"substituters"`
	SyncBeforeRegistering                 ConfigField[bool]              `json:"sync-before-registering"`
	System                                ConfigField[string]            `json:"system"`
	SystemFeatures                        ConfigField[[]string]          `json:"system-features"`
	TarballTTL                            ConfigField[int]               `json:"tarball-ttl"`
	Timeout                               ConfigField[int]               `json:"timeout"`
	TraceFunctionCalls                    ConfigField[bool]              `json:"trace-function-calls"`
	TraceVerbose                          ConfigField[bool]              `json:"trace-verbose"`
	TrustedPublicKeys                     ConfigField[[]string]          `json:"trusted-public-keys"`
	TrustedSubstituters                   ConfigField[[]string]          `json:"trusted-substituters"`
	TrustedUsers                          ConfigField[[]string]          `json:"trusted-users"`
	UpgradeNixStorePathURL                ConfigField[string]            `json:"upgrade-nix-store-path-url"`
	UseCaseHack                           ConfigField[bool]              `json:"use-case-hack"`
	UseRegistries                         ConfigField[bool]              `json:"use-registries"`
	UseSqliteWAL                          ConfigField[bool]              `json:"use-sqlite-wal"`
	UseXDGBaseDirectories                 ConfigField[bool]              `json:"use-xdg-base-directories"`
	UserAgentSuffix                       ConfigField[string]            `json:"user-agent-suffix"`
	WarnDirty                             ConfigField[bool]              `json:"warn-dirty"`
}

// ConfigField is a Nix configuration setting.
type ConfigField[T any] struct {
	Aliases             []string `json:"aliases"`
	DefaultValue        T        `json:"defaultValue"`
	Description         string   `json:"description"`
	DocumentDefault     bool     `json:"documentDefault"`
	ExperimentalFeature string   `json:"experimentalFeature"`
	Value               T        `json:"value"`
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
