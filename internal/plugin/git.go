// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"go.jetify.com/devbox/nix/flake"
	"go.jetify.com/pkg/filecache"
)

var gitCache = filecache.New[[]byte]("devbox/plugin/git")

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

// gitClones holds one clone per repo/ref/rev for the life of the process, so
// that reading several files from a git plugin (plugin.json plus the files it
// creates) clones the repo once instead of once per file. A failed clone is
// remembered too, so an unreachable remote is only tried once per process.
var gitClones = struct {
	sync.Mutex
	clones map[string]gitClone
}{clones: map[string]gitClone{}}

type gitClone struct {
	dir string
	err error
}

// cloneGitRepo clones the plugin's repo into a new temporary directory and
// returns its path. Tests override it.
var cloneGitRepo = (*gitPlugin).clone

// RemoveGitClones deletes the git plugin clones made by this process.
func RemoveGitClones() {
	gitClones.Lock()
	defer gitClones.Unlock()
	for key, c := range gitClones.clones {
		if c.dir != "" {
			_ = os.RemoveAll(c.dir)
		}
		delete(gitClones.clones, key)
	}
}

// cloneDir returns a clone of the plugin's repo at its ref/rev, cloning it if
// this process hasn't already. Plugins that differ only by dir share a clone.
func (p *gitPlugin) cloneDir() (string, error) {
	key := strings.Join([]string{p.getBaseURL(), p.ref.Ref, p.ref.Rev}, "\x00")

	gitClones.Lock()
	defer gitClones.Unlock()
	c, ok := gitClones.clones[key]
	if !ok {
		c.dir, c.err = cloneGitRepo(p)
		gitClones.clones[key] = c
	}
	return c.dir, c.err
}

func (p *gitPlugin) clone() (string, error) {
	tempDir, err := os.MkdirTemp("", "devbox-git-plugin-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	baseURL := p.getBaseURL()

	cloneArgs := []string{"clone"}
	// A shallow clone only has the branch tip, so when a specific rev is
	// requested we do a full clone and check it out below.
	if p.ref.Rev == "" {
		cloneArgs = append(cloneArgs, "--depth", "1")
		if p.ref.Ref != "" {
			cloneArgs = append(cloneArgs, "--branch", p.ref.Ref)
		}
	}
	cloneArgs = append(cloneArgs, baseURL, tempDir)
	cloneCmd := exec.Command("git", cloneArgs...)

	if isSSHURL(baseURL) {
		gitSSHCommand := os.Getenv("GIT_SSH_COMMAND")
		if gitSSHCommand == "" {
			gitSSHCommand = "ssh -o StrictHostKeyChecking=accept-new"
		}
		cloneCmd.Env = append(os.Environ(), "GIT_SSH_COMMAND="+gitSSHCommand)
	}

	output, err := cloneCmd.CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to clone repository %s: %w\nOutput: %s", p.ref.URL, err, string(output))
	}

	if p.ref.Rev != "" {
		checkoutCmd := exec.Command("git", "checkout", p.ref.Rev)
		checkoutCmd.Dir = tempDir
		output, err := checkoutCmd.CombinedOutput()
		if err != nil {
			_ = os.RemoveAll(tempDir)
			return "", fmt.Errorf("failed to checkout revision %s: %w\nOutput: %s", p.ref.Rev, err, string(output))
		}
	}
	return tempDir, nil
}

// readFromClone reads subpath from the plugin's directory in a clone of its
// repo.
func (p *gitPlugin) readFromClone(subpath string) ([]byte, error) {
	cloneDir, err := p.cloneDir()
	if err != nil {
		return nil, err
	}

	// Read file from repository root or specified directory
	filePath := filepath.Join(cloneDir, p.ref.Dir, subpath)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	return content, nil
}

// isSSHURL checks if the given URL is an SSH URL.
// SSH URLs can be in formats:
//   - ssh://user@host/path
//   - git@host:path
//   - ssh://git@host/path
func isSSHURL(url string) bool {
	url = strings.TrimSpace(url)
	// Check for explicit ssh:// protocol
	if strings.HasPrefix(url, "ssh://") {
		return true
	}
	// Check for git@host:path format (SCP-like syntax)
	// This format uses colon after host, not port number
	if strings.HasPrefix(url, "git@") && strings.Contains(url, ":") {
		// Make sure it's not an HTTPS URL with port (e.g., https://git@host:443/path)
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			return true
		}
	}
	return false
}

func isBranchName(ref string) bool {
	// Full commit hashes are 40 hex characters
	if len(ref) == 40 {
		for _, c := range ref {
			isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
			if !isHex {
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
	return fmt.Sprintf("%s-%s-%s-%s", p.ref.URL, p.ref.Rev, p.ref.Ref, p.ref.Dir)
}

func (p *gitPlugin) FileContent(subpath string) ([]byte, error) {
	// Content at a full commit hash is cached indefinitely. Anything else is
	// cached for 24 hours.
	ttl, err := remoteCacheTTL(p.ref.Rev)
	if err != nil {
		return nil, err
	}
	cacheKey := p.LockfileKey() + "/" + subpath + "/" + ttl.String()
	return getOrFetch(gitCache, cacheKey, ttl, func() ([]byte, error) {
		return p.readFromClone(subpath)
	})
}

func (p *gitPlugin) LockfileKey() string {
	return p.ref.String()
}
