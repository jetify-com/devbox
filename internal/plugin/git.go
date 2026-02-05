// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.jetify.com/devbox/nix/flake"
)

type gitPlugin struct {
	ref  *flake.Ref
	name string
}

// newGitPlugin creates a Git plugin from a flake reference.
// It uses git clone to fetch the repository.
func newGitPlugin(ref flake.Ref) (*gitPlugin, error) {
	if ref.Type != flake.TypeGit {
		return nil, fmt.Errorf("expected git flake reference, got %s", ref.Type)
	}

	name := generateGitPluginName(ref)

	return &gitPlugin{
		ref:  &ref,
		name: name,
	}, nil
}

func generateGitPluginName(ref flake.Ref) string {
	// Extract repository name from URL and append directory if specified
	url := ref.URL
	if url == "" {
		return "unknown.git"
	}

	// Remove query parameters to get clean URL
	if strings.Contains(url, "?") {
		url = strings.Split(url, "?")[0]
	}

	url = strings.TrimSuffix(url, ".git")

	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "unknown.git"
	}

	// Use last two path components (e.g., "owner/repo")
	repoParts := parts[len(parts)-2:]

	name := strings.Join(repoParts, ".")
	name = strings.ReplaceAll(name, "/", ".")

	// Append directory to make name unique when multiple plugins
	// from same repo are used
	if ref.Dir != "" {
		dirName := strings.ReplaceAll(ref.Dir, "/", ".")
		name = name + "." + dirName
	}

	return name
}

// getBaseURL extracts the base Git URL without query parameters.
// Query parameters like ?dir=path are used by Nix flakes but not by git clone.
func (p *gitPlugin) getBaseURL() string {
	baseURL := p.ref.URL
	if strings.Contains(baseURL, "?") {
		baseURL = strings.Split(baseURL, "?")[0]
	}
	return baseURL
}

func (p *gitPlugin) Fetch() ([]byte, error) {
	content, err := p.FileContent("plugin.json")
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (p *gitPlugin) cloneAndRead(subpath string) ([]byte, error) {
	tempDir, err := os.MkdirTemp("", "devbox-git-plugin-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone repository using base URL without query parameters
	baseURL := p.getBaseURL()

	// For branch names, use --branch to clone that specific branch
	// For commit hashes, clone default branch then fetch the specific commit
	var cloneCmd *exec.Cmd
	if p.ref.Rev != "" && isBranchName(p.ref.Rev) {
		cloneCmd = exec.Command("git", "clone", "--depth", "1", "--branch", p.ref.Rev, baseURL, tempDir)
	} else {
		cloneCmd = exec.Command("git", "clone", "--depth", "1", baseURL, tempDir)
	}

	output, err := cloneCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository %s: %w\nOutput: %s", p.ref.URL, err, string(output))
	}

	// Checkout specific commit if revision is a commit hash
	if p.ref.Rev != "" && !isBranchName(p.ref.Rev) {
		// Fetch the specific commit
		fetchCmd := exec.Command("git", "fetch", "--depth", "1", "origin", p.ref.Rev)
		fetchCmd.Dir = tempDir
		output, err := fetchCmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch revision %s: %w\nOutput: %s", p.ref.Rev, err, string(output))
		}

		checkoutCmd := exec.Command("git", "checkout", p.ref.Rev)
		checkoutCmd.Dir = tempDir
		output, err = checkoutCmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to checkout revision %s: %w\nOutput: %s", p.ref.Rev, err, string(output))
		}
	}

	// Read file from repository root or specified directory
	filePath := filepath.Join(tempDir, subpath)
	if p.ref.Dir != "" {
		filePath = filepath.Join(tempDir, p.ref.Dir, subpath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return content, nil
}

func isBranchName(ref string) bool {
	// Full commit hashes are 40 hex characters
	if len(ref) == 40 {
		for _, c := range ref {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return true
			}
		}
		return false
	}
	return true
}

func (p *gitPlugin) CanonicalName() string {
	return p.name
}

// Hash returns a unique hash for this plugin including directory.
// This ensures plugins from the same repo with different dirs are unique.
func (p *gitPlugin) Hash() string {
	if p.ref.Dir != "" {
		return fmt.Sprintf("%s-%s-%s", p.ref.URL, p.ref.Rev, p.ref.Dir)
	}
	return fmt.Sprintf("%s-%s", p.ref.URL, p.ref.Rev)
}

func (p *gitPlugin) FileContent(subpath string) ([]byte, error) {
	return p.cloneAndRead(subpath)
}

func (p *gitPlugin) LockfileKey() string {
	return p.ref.String()
}
