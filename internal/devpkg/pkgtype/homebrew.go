package pkgtype

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	HomebrewScheme = "homebrew"
	HomebrewPrefix = HomebrewScheme + ":"
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

// IsInstalled reports whether the `brew` CLI is available on the user's PATH.
func (h *Homebrew) IsInstalled() bool {
	_, err := exec.LookPath("brew")
	return err == nil
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
		if line != "" && !strings.HasPrefix(line, "==>") {
			results = append(results, line)
		}
	}
	return results, nil
}

func (h *Homebrew) install(ctx context.Context, formula string) error {
	cmd := exec.CommandContext(ctx, "brew", "install", formula)
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
	if !h.IsInstalled() {
		return nil, fmt.Errorf(
			"homebrew is required to use homebrew: packages, but `brew` was not " +
				"found on your PATH. Install it from https://brew.sh",
		)
	}
	return exec.CommandContext(ctx, "brew", args...).Output()
}
