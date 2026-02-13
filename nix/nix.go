package nix

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.jetify.com/devbox/internal/redact"
	"golang.org/x/mod/semver"
)

// Default is the default Nix installation.
var Default = &Nix{}

// Command creates an arbitrary command using the Nix executable found in $PATH.
// It's the same as calling [Nix.Command] on the default Nix installation.
func Command(args ...any) *Cmd {
	return Default.Command(args...)
}

// System calls [Nix.System] on the default Nix installation.
func System() string {
	return Default.System()
}

// Version calls [Nix.Version] on the default Nix installation.
func Version() string {
	return Default.Version()
}

// AtLeast reports if the default Nix installation's version is equal to or
// newer than the given version. It returns false if it cannot determine the
// Nix version.
func AtLeast(version string) bool {
	info, _ := Default.Info()
	return info.AtLeast(version)
}

// Nix provides an interface for interacting with Nix. The zero-value is valid
// and uses the first Nix executable found in $PATH.
type Nix struct {
	// Path is the absolute path to the nix executable. If it is empty,
	// nix commands use the executable found in $PATH.
	Path     string
	lookPath atomic.Pointer[string]

	// ExtraArgs are command line arguments to pass to every Nix command.
	ExtraArgs Args

	info     Info
	infoErr  error
	infoOnce sync.Once

	// Logger logs information at [slog.LevelDebug] about Nix command
	// starts and exits. If nil, it defaults to [slog.Default].
	Logger *slog.Logger
}

// resolvePath resolves the path to the Nix executable. It returns n.Path if it
// is non-empty and a valid executable. Otherwise it searches for a nix
// executable in $PATH and common installation directories.
func (n *Nix) resolvePath() (string, error) {
	if n.Path != "" {
		return exec.LookPath(n.Path) // verify it's an executable.
	}

	// Re-use the cached path if we've already found Nix before.
	cached := n.lookPath.Load()
	if cached != nil && *cached != "" {
		return *cached, nil
	}

	_, _ = SourceProfile()
	path, pathErr := exec.LookPath("nix")
	if pathErr == nil {
		n.lookPath.Store(&path)
		return path, nil
	}

	try := []string{
		"/nix/var/nix/profiles/default/bin/nix",
		"/run/current-system/sw/bin",
	}
	for _, path := range try {
		stat, err := os.Stat(path)
		if err == nil {
			// Is it executable and not a directory?
			m := stat.Mode()
			if !m.IsDir() && m.Perm()&0o111 != 0 {
				n.lookPath.Store(&path)
				return path, nil
			}
		}
	}
	return "", pathErr
}

func (n *Nix) logger() *slog.Logger {
	if n.Logger == nil {
		return slog.Default()
	}
	return n.Logger
}

// System returns the system from [Nix.Info] or an empty string if there was an
// error.
func (n *Nix) System() string {
	info, _ := n.Info()
	return info.System
}

// Version returns the version from [Nix.Info] or an empty string if there was
// an error.
func (n *Nix) Version() string {
	info, _ := n.Info()
	return info.Version
}

// Info returns Nix version information. It caches the result after the first
// call, which means it won't reflect any configuration changes to Nix. Create a
// new Nix instance to retrieve uncached information.
func (n *Nix) Info() (Info, error) {
	// Create the command first, which will catch any errors finding the Nix
	// executable outside of the once. This allows us to retry after
	// installing Nix.
	cmd := n.Command("--version", "--debug")
	if cmd.err != nil {
		return Info{}, cmd.err
	}

	n.infoOnce.Do(func() {
		out, err := cmd.Output(context.Background())
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) && len(exitErr.Stderr) != 0 {
				n.infoErr = redact.Errorf("nix command: %s: %q: %v", redact.Safe(cmd), exitErr.Stderr, err)
				return
			}
			n.infoErr = redact.Errorf("nix command: %s: %v", redact.Safe(cmd), err)
			return
		}
		n.info, n.infoErr = parseInfo(out)
	})
	return n.info, n.infoErr
}

// All major Nix versions supported by Devbox.
const (
	Version2_12 = "2.12.0"
	Version2_13 = "2.13.0"
	Version2_14 = "2.14.0"
	Version2_15 = "2.15.0"
	Version2_16 = "2.16.0"
	Version2_17 = "2.17.0"
	Version2_18 = "2.18.0"
	Version2_19 = "2.19.0"
	Version2_20 = "2.20.0"
	Version2_21 = "2.21.0"
	Version2_22 = "2.22.0"
	Version2_23 = "2.23.0"
	Version2_24 = "2.24.0"
	Version2_25 = "2.25.0"

	MinVersion = Version2_18
)

// versionRegexp matches the first line of "nix --version" output.
//
// The semantic component is sourced from <https://semver.org/#is-there-a-suggested-regular-expression-regex-to-check-a-semver-string>.
// It's been modified to tolerate Nix prerelease versions, which don't have a
// hyphen before the prerelease component and contain underscores.
var versionRegexp = regexp.MustCompile(`^(.+) \(.+\) ((?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:(?:-|pre)(?P<prerelease>(?:0|[1-9]\d*|\d*[_a-zA-Z-][_0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[_a-zA-Z-][_0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?)$`)

// preReleaseRegexp matches Nix prerelease version strings, which are not valid
// semvers.
var preReleaseRegexp = regexp.MustCompile(`pre(?P<date>[0-9]+)_(?P<commit>[a-f0-9]{4,40})$`)

// Info contains information about a Nix installation.
type Info struct {
	// Name identifies the Nix implementation. It is usually "nix" but may
	// also be a fork like "lix".
	Name string

	// Version is the semantic Nix version string.
	Version string

	// System is the Nix system tuple. It follows the pattern <arch>-<os>
	// and does not use the same values as GOOS or GOARCH. Note that the Nix
	// system is configurable and may not represent the actual operating
	// system or architecture.
	System string

	// ExtraSystems are other systems that the current machine supports.
	// Usually set by the extra-platforms setting in nix.conf.
	ExtraSystems []string

	// Features are the capabilities that the Nix binary was compiled with.
	Features []string

	// SystemConfig is the path to the Nix system configuration file,
	// usually /etc/nix/nix.conf.
	SystemConfig string

	// UserConfigs is a list of paths to the user's Nix configuration files.
	UserConfigs []string

	// StoreDir is the path to the Nix store directory, usually /nix/store.
	StoreDir string

	// StateDir is the path to the Nix state directory, usually
	// /nix/var/nix.
	StateDir string

	// DataDir is the path to the Nix data directory, usually somewhere
	// within the Nix store. This field is empty for Nix versions <= 2.12.
	DataDir string
}

func parseInfo(data []byte) (Info, error) {
	// Example nix --version --debug output from Nix versions 2.12 to 2.21.
	// Version 2.12 omits the data directory, but they're otherwise
	// identical.
	//
	// See https://github.com/NixOS/nix/blob/5b9cb8b3722b85191ee8cce8f0993170e0fc234c/src/libmain/shared.cc#L284-L305
	//
	// nix (Nix) 2.21.2
	// System type: aarch64-darwin
	// Additional system types: x86_64-darwin
	// Features: gc, signed-caches
	// System configuration file: /etc/nix/nix.conf
	// User configuration files: /Users/nobody/.config/nix/nix.conf:/etc/xdg/nix/nix.conf
	// Store directory: /nix/store
	// State directory: /nix/var/nix
	// Data directory: /nix/store/m0ns07v8by0458yp6k30rfq1rs3kaz6g-nix-2.21.2/share

	info := Info{}
	if len(data) == 0 {
		return info, redact.Errorf("empty nix --version output")
	}

	lines := strings.Split(string(data), "\n")
	matches := versionRegexp.FindStringSubmatch(lines[0])
	if len(matches) < 3 {
		return info, redact.Errorf("parse nix version: %s", redact.Safe(lines[0]))
	}
	info.Name = matches[1]
	info.Version = matches[2]
	for _, line := range lines {
		name, value, found := strings.Cut(line, ": ")
		if !found {
			continue
		}

		switch name {
		case "System type":
			info.System = value
		case "Additional system types":
			info.ExtraSystems = strings.Split(value, ", ")
		case "Features":
			info.Features = strings.Split(value, ", ")
		case "System configuration file":
			info.SystemConfig = value
		case "User configuration files":
			info.UserConfigs = strings.Split(value, ":")
		case "Store directory":
			info.StoreDir = value
		case "State directory":
			info.StateDir = value
		case "Data directory":
			info.DataDir = value
		}
	}
	return info, nil
}

// AtLeast returns true if i.Version is >= version per semantic versioning. It
// always returns false if i.Version is empty or invalid, such as when the
// current Nix version cannot be parsed. It panics if version is an invalid
// semver.
func (i Info) AtLeast(version string) bool {
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	if !semver.IsValid(version) {
		panic(fmt.Sprintf("nix.atLeast: invalid version %q", version[1:]))
	}
	if semver.IsValid("v" + i.Version) {
		return semver.Compare("v"+i.Version, version) >= 0
	}

	// If the version isn't a valid semver, check to see if it's a
	// prerelease (e.g., 2.23.0pre20240526_7de033d6) and coerce it to a
	// valid version (2.23.0-pre.20240526+7de033d6) so we can compare it.
	prerelease := preReleaseRegexp.ReplaceAllString(i.Version, "-pre.$date+$commit")
	return semver.Compare("v"+prerelease, version) >= 0
}

// sourceProfileMutex guards against multiple goroutines attempting to source
// the Nix profile scripts concurrently.
var sourceProfileMutex sync.Mutex

// SourceProfile adds environment variables from the Nix profile shell scripts
// to the current process's environment. This ensures that PATH contains the nix
// bin directory and that NIX_PROFILES and NIX_SSL_CERT_FILE are set.
//
// For properly configured Nix installations, the user's login shell handles
// sourcing the profile and SourceProfile has no effect.
func SourceProfile() (sourced bool, err error) {
	if profileSourced() {
		return false, nil
	}
	sourceProfileMutex.Lock()
	defer sourceProfileMutex.Unlock()

	if profileSourced() {
		return false, nil
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	shell, _ = exec.LookPath("sh")
	if shell == "" {
		shell = "/bin/sh"
	}

	trySource := func(path string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		wantEnv := map[string]bool{
			"NIX_PROFILES":              true,
			"NIX_SSL_CERT_FILE":         true,
			"PATH":                      true,
			"XDG_DATA_DIRS":             true,
			"__ETC_PROFILE_NIX_SOURCED": true,
		}
		script := fmt.Sprintf(". \"%s\"\n", path)
		for name := range wantEnv {
			script += fmt.Sprintf("echo %s=\"$%[1]s\"\n", name)
		}

		cmd := exec.CommandContext(ctx, shell, "-e", "-c", script)
		stdout, err := cmd.Output()
		if err != nil {
			return err
		}
		for _, line := range strings.Split(string(stdout), "\n") {
			name, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			if wantEnv[name] {
				err = os.Setenv(name, value)
				if err != nil {
					return err
				}
				delete(wantEnv, name)
			}
		}
		return nil
	}

	for _, path := range profilePaths() {
		err = trySource(path)
		if err == nil {
			return true, nil
		}
	}
	return false, fmt.Errorf("unable to source Nix profile")
}

// profileSourced checks if the Nix profile shell scripts (such as
// /nix/var/nix/profiles/default/etc/profile.d/nix-daemon.sh) have already been
// successfully sourced.
func profileSourced() bool {
	// Check if we're already in a Nix environment. Use NIX_PROFILES instead
	// of __ETC_PROFILE_NIX_SOURCED because it's set for single-user
	// installs and on NixOS (whereas __ETC_PROFILE_NIX_SOURCED is not).
	_, ok := os.LookupEnv("NIX_PROFILES")
	return ok
}

// profilePaths returns the paths where the Nix profile shell scripts might be.
// None of the paths are guaranteed to be readable or even exist.
func profilePaths() []string {
	// os.UserHomeDir only checks $HOME, user.Current reads /etc/passwd or
	// uses libc. This can help when running in an isolated environment
	// where $HOME isn't set.
	home, _ := os.UserHomeDir()
	if home == "" {
		if u, err := user.Current(); err == nil {
			home = u.HomeDir
		}
	}
	if home == "" {
		// Might as well check the root home directory if we've got
		// nothing else.
		home = "/root"
	}
	xdgState := os.Getenv("XDG_STATE_HOME")
	if xdgState == "" {
		xdgState = filepath.Join(home, ".local/state")
	}

	dirs := make([]string, 0, 5)
	if nixExe, err := exec.LookPath("nix"); err == nil {
		dirs = append(dirs, filepath.Clean(nixExe+"/../../etc/profile.d"))
	}
	if !slices.Contains(dirs, "/nix/var/nix/profiles/default/etc/profile.d") {
		dirs = append(dirs, "/nix/var/nix/profiles/default/etc/profile.d")
	}
	dirs = append(dirs,
		filepath.Join(home, ".nix-profile/etc/profile.d"),
		filepath.Join(xdgState, "nix/profile/etc/profile.d"),
		filepath.Join(xdgState, "nix/profiles/profile/etc/profile.d"),
	)

	// Try sourcing scripts in the following order:
	//
	//  1. nix-daemon.sh: because nix.sh is a no-op when $USER isn't set
	//     (this happens in containers).
	//  2. nix-daemon.fish: same, but for fish users.
	//  3. nix.sh, nix.fish: for old single-user installs.
	files := make([]string, 0, len(dirs)*4)
	for _, dir := range dirs {
		files = append(files, filepath.Join(dir, "nix-daemon.sh"))
	}
	for _, dir := range dirs {
		files = append(files, filepath.Join(dir, "nix-daemon.fish"))
	}
	for _, dir := range dirs {
		files = append(files, filepath.Join(dir, "nix.sh"))
	}
	for _, dir := range dirs {
		files = append(files, filepath.Join(dir, "nix.fish"))
	}
	return files
}
