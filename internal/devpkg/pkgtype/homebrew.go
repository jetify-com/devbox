package pkgtype

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	HomebrewScheme = "homebrew"
	HomebrewPrefix = HomebrewScheme + ":"

	// homebrewInstallScriptURL is the official Homebrew installer. It supports
	// both macOS and Linux.
	homebrewInstallScriptURL = "https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh"
)

// IsHomebrew returns true if the package string refers to a Homebrew formula,
// e.g. "homebrew:python@3.10".
func IsHomebrew(s string) bool {
	return strings.HasPrefix(s, HomebrewPrefix)
}

// HomebrewClient returns a client that shells out to the `brew` CLI to install
// and inspect Homebrew formulae.
func HomebrewClient() *Homebrew {
	return &Homebrew{}
}

type Homebrew struct{}

// brewInfo is a partial representation of the JSON returned by
// `brew info --json=v2 <formula>`. We only decode the fields we need.
type brewInfo struct {
	Formulae []brewFormula `json:"formulae"`
}

type brewFormula struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	// VersionedFormulae lists the names of the versioned variants of this
	// formula (e.g. python -> ["python@3.13", "python@3.12", ...]). It is
	// empty for formulae that don't ship versioned variants.
	VersionedFormulae []string `json:"versioned_formulae"`
}

// brewKnownPaths are the default locations Homebrew installs the `brew` binary
// on Linux and macOS. We check these so that devbox can find brew immediately
// after installing it, even before the user has updated their shell PATH.
func brewKnownPaths() []string {
	paths := []string{
		"/home/linuxbrew/.linuxbrew/bin/brew", // Linux (default)
		"/opt/homebrew/bin/brew",              // macOS (Apple Silicon)
		"/usr/local/bin/brew",                 // macOS (Intel)
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".linuxbrew", "bin", "brew"))
	}
	return paths
}

// brewPath returns the path to the `brew` executable, looking first on PATH and
// then in the default install locations. It returns "" if brew is not found.
func (h *Homebrew) brewPath() string {
	if path, err := exec.LookPath("brew"); err == nil {
		return path
	}
	for _, path := range brewKnownPaths() {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// IsInstalled reports whether the `brew` CLI is available, either on PATH or in
// one of the default Homebrew install locations.
func (h *Homebrew) IsInstalled() bool {
	return h.brewPath() != ""
}

// Bootstrap installs Homebrew itself using the official install script. It runs
// non-interactively (the caller is responsible for confirming with the user
// first). It works on both macOS and Linux. Output is streamed to w.
func (h *Homebrew) Bootstrap(ctx context.Context, w io.Writer) error {
	script, err := h.downloadInstallScript(ctx)
	if err != nil {
		return err
	}
	defer os.Remove(script)

	cmd := exec.CommandContext(ctx, "/bin/bash", script)
	// NONINTERACTIVE tells the Homebrew installer not to prompt; we've already
	// confirmed with the user (or are running non-interactively).
	cmd.Env = append(os.Environ(), "NONINTERACTIVE=1")
	cmd.Stdin = os.Stdin
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install homebrew: %w", err)
	}

	if !h.IsInstalled() {
		return fmt.Errorf(
			"homebrew installation completed but `brew` could not be found in the " +
				"expected locations",
		)
	}

	// Make brew (and its installed formulae) available to the rest of this
	// process by adding its bin directory to PATH, similar to sourcing
	// `brew shellenv`.
	if brewBin := filepath.Dir(h.brewPath()); brewBin != "" {
		path := os.Getenv("PATH")
		if !strings.Contains(path, brewBin) {
			_ = os.Setenv("PATH", brewBin+string(os.PathListSeparator)+path)
		}
	}
	return nil
}

func (h *Homebrew) downloadInstallScript(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, homebrewInstallScriptURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download homebrew installer: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"failed to download homebrew installer: unexpected status %s", resp.Status)
	}

	tmpFile, err := os.CreateTemp("", "homebrew-install-*.sh")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}
	return tmpFile.Name(), nil
}

// VersionedFormulae returns the names of the versioned variants of the given
// base formula (e.g. "python" -> ["python@3.13", "python@3.12"]). It returns an
// empty slice for formulae that don't support versioned formulae.
func (h *Homebrew) VersionedFormulae(ctx context.Context, formula string) ([]string, error) {
	info, err := h.info(ctx, formula)
	if err != nil {
		return nil, err
	}
	if len(info.Formulae) == 0 {
		return nil, fmt.Errorf("homebrew formula %q not found", formula)
	}
	return info.Formulae[0].VersionedFormulae, nil
}

// EnsureInstalled installs the formula if it isn't already installed and
// returns the directories that should be added to PATH (e.g. its bin dir).
func (h *Homebrew) EnsureInstalled(ctx context.Context, formula string) ([]string, error) {
	prefix, err := h.prefix(ctx, formula)
	if err != nil {
		return nil, err
	}
	// `brew --prefix <formula>` returns the opt path even when the formula is
	// not installed yet, so we check whether that path exists on disk to decide
	// if we need to install.
	if _, statErr := os.Stat(prefix); statErr != nil {
		if err := h.install(ctx, formula); err != nil {
			return nil, err
		}
	}

	paths := []string{}
	binPath := filepath.Join(prefix, "bin")
	if _, err := os.Stat(binPath); err == nil {
		paths = append(paths, binPath)
	}
	return paths, nil
}

// Search returns the names of formulae that match the given query.
func (h *Homebrew) Search(ctx context.Context, query string) ([]string, error) {
	out, err := h.run(ctx, "search", "--formula", query)
	if err != nil {
		return nil, err
	}
	results := []string{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "==>") {
			continue
		}
		// `brew search` may print multiple formula names per line, separated by
		// whitespace, so split each line into individual formulae.
		results = append(results, strings.Fields(line)...)
	}
	return results, nil
}

func (h *Homebrew) install(ctx context.Context, formula string) error {
	cmd := exec.CommandContext(ctx, h.brewPath(), "install", formula)
	// Stream brew's output so the user can follow along with the install.
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install homebrew formula %q: %w", formula, err)
	}
	return nil
}

func (h *Homebrew) prefix(ctx context.Context, formula string) (string, error) {
	out, err := h.run(ctx, "--prefix", formula)
	if err != nil {
		return "", fmt.Errorf("homebrew formula %q not found: %w", formula, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (h *Homebrew) info(ctx context.Context, formula string) (*brewInfo, error) {
	out, err := h.run(ctx, "info", "--json=v2", formula)
	if err != nil {
		return nil, err
	}
	info := &brewInfo{}
	if err := json.Unmarshal(out, info); err != nil {
		return nil, fmt.Errorf("failed to parse homebrew info for %q: %w", formula, err)
	}
	return info, nil
}

func (h *Homebrew) run(ctx context.Context, args ...string) ([]byte, error) {
	brew := h.brewPath()
	if brew == "" {
		return nil, fmt.Errorf(
			"homebrew is required to use homebrew: packages, but `brew` was not " +
				"found. Install it from https://brew.sh",
		)
	}
	return exec.CommandContext(ctx, brew, args...).Output()
}
