// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"testing"

	"go.jetify.com/devbox/nix/flake"
)

func TestGitPlugin(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected *gitPlugin
	}{
		{
			name: "basic git plugin",
			ref:  "git+https://github.com/jetify-com/devbox-plugins.git",
			expected: &gitPlugin{
				ref: &flake.Ref{
					Type: flake.TypeGit,
					URL:  "https://github.com/jetify-com/devbox-plugins.git",
				},
				name: "jetify-com.devbox-plugins",
			},
		},
		{
			name: "git plugin with ref",
			ref:  "git+https://github.com/jetify-com/devbox-plugins.git?ref=main",
			expected: &gitPlugin{
				ref: &flake.Ref{
					Type: flake.TypeGit,
					URL:  "https://github.com/jetify-com/devbox-plugins.git",
					Ref:  "main",
				},
				name: "jetify-com.devbox-plugins",
			},
		},
		{
			name: "git plugin with rev",
			ref:  "git+https://github.com/jetify-com/devbox-plugins.git?rev=abc123",
			expected: &gitPlugin{
				ref: &flake.Ref{
					Type: flake.TypeGit,
					URL:  "https://github.com/jetify-com/devbox-plugins.git",
					Rev:  "abc123",
				},
				name: "jetify-com.devbox-plugins",
			},
		},
		{
			name: "git plugin with directory",
			ref:  "git+https://github.com/jetify-com/devbox-plugins.git?dir=mongodb",
			expected: &gitPlugin{
				ref: &flake.Ref{
					Type: flake.TypeGit,
					URL:  "https://github.com/jetify-com/devbox-plugins.git?dir=mongodb",
					Dir:  "mongodb",
				},
				name: "jetify-com.devbox-plugins.mongodb",
			},
		},
		{
			name: "git plugin with directory and ref",
			ref:  "git+https://github.com/jetify-com/devbox-plugins.git?dir=mongodb&ref=my-branch",
			expected: &gitPlugin{
				ref: &flake.Ref{
					Type: flake.TypeGit,
					URL:  "https://github.com/jetify-com/devbox-plugins.git?dir=mongodb",
					Dir:  "mongodb",
					Ref:  "my-branch",
				},
				name: "jetify-com.devbox-plugins.mongodb",
			},
		},
		{
			name: "git plugin with subgroups",
			ref:  "git+https://gitlab.com/group/subgroup/repo.git",
			expected: &gitPlugin{
				ref: &flake.Ref{
					Type: flake.TypeGit,
					URL:  "https://gitlab.com/group/subgroup/repo.git",
				},
				name: "subgroup.repo",
			},
		},
		{
			name: "git plugin with SSH URL",
			ref:  "git+ssh://git@github.com/jetify-com/devbox-plugins.git",
			expected: &gitPlugin{
				ref: &flake.Ref{
					Type: flake.TypeGit,
					URL:  "ssh://git@github.com/jetify-com/devbox-plugins.git",
				},
				name: "jetify-com.devbox-plugins",
			},
		},
		{
			name: "git plugin with file URL",
			ref:  "git+file:///tmp/local-repo.git",
			expected: &gitPlugin{
				ref: &flake.Ref{
					Type: flake.TypeGit,
					URL:  "file:///tmp/local-repo.git",
				},
				name: "tmp.local-repo",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := flake.ParseRef(tt.ref)
			if err != nil {
				t.Fatalf("Failed to parse ref %q: %v", tt.ref, err)
			}

			plugin, err := newGitPlugin(ref)
			if err != nil {
				t.Fatalf("Failed to create Git plugin: %v", err)
			}

			if plugin.ref.Type != tt.expected.ref.Type {
				t.Errorf("Expected type %q, got %q", tt.expected.ref.Type, plugin.ref.Type)
			}
			if plugin.ref.URL != tt.expected.ref.URL {
				t.Errorf("Expected URL %q, got %q", tt.expected.ref.URL, plugin.ref.URL)
			}
			if plugin.ref.Ref != tt.expected.ref.Ref {
				t.Errorf("Expected ref %q, got %q", tt.expected.ref.Ref, plugin.ref.Ref)
			}
			if plugin.ref.Rev != tt.expected.ref.Rev {
				t.Errorf("Expected rev %q, got %q", tt.expected.ref.Rev, plugin.ref.Rev)
			}
			if plugin.ref.Dir != tt.expected.ref.Dir {
				t.Errorf("Expected dir %q, got %q", tt.expected.ref.Dir, plugin.ref.Dir)
			}
			if plugin.name != tt.expected.name {
				t.Errorf("Expected name %q, got %q", tt.expected.name, plugin.name)
			}
		})
	}
}

func TestGenerateGitPluginName(t *testing.T) {
	tests := []struct {
		name     string
		ref      flake.Ref
		expected string
	}{
		{
			name: "github repository",
			ref: flake.Ref{
				URL: "https://github.com/jetify-com/devbox-plugins.git",
			},
			expected: "jetify-com.devbox-plugins",
		},
		{
			name: "gitlab repository with subgroups",
			ref: flake.Ref{
				URL: "https://gitlab.com/group/subgroup/repo.git",
			},
			expected: "subgroup.repo",
		},
		{
			name: "repository without .git suffix",
			ref: flake.Ref{
				URL: "https://github.com/jetify-com/devbox-plugins",
			},
			expected: "jetify-com.devbox-plugins",
		},
		{
			name: "repository with single path component",
			ref: flake.Ref{
				URL: "https://github.com/repo",
			},
			expected: "github.com.repo",
		},
		{
			name: "SSH repository",
			ref: flake.Ref{
				URL: "ssh://git@github.com/jetify-com/devbox-plugins.git",
			},
			expected: "jetify-com.devbox-plugins",
		},
		{
			name: "file repository",
			ref: flake.Ref{
				URL: "file:///tmp/local-repo.git",
			},
			expected: "tmp.local-repo",
		},
		{
			name: "repository with directory",
			ref: flake.Ref{
				URL: "https://github.com/jetify-com/devbox-plugins.git",
				Dir: "mongodb",
			},
			expected: "jetify-com.devbox-plugins.mongodb",
		},
		{
			name: "repository with nested directory",
			ref: flake.Ref{
				URL: "https://github.com/jetify-com/devbox-plugins.git",
				Dir: "plugins/python",
			},
			expected: "jetify-com.devbox-plugins.plugins.python",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateGitPluginName(tt.ref)
			if result != tt.expected {
				t.Errorf("Expected name %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGitPluginURL(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		subpath  string
		expected string
	}{
		{
			name:     "basic plugin.json",
			ref:      "git+https://github.com/jetify-com/devbox-plugins.git",
			subpath:  "plugin.json",
			expected: "plugin.json",
		},
		{
			name:     "plugin with directory",
			ref:      "git+https://github.com/jetify-com/devbox-plugins.git?dir=mongodb",
			subpath:  "plugin.json",
			expected: "mongodb/plugin.json",
		},
		{
			name:     "plugin with ref",
			ref:      "git+https://github.com/jetify-com/devbox-plugins.git?ref=main",
			subpath:  "plugin.json",
			expected: "plugin.json",
		},
		{
			name:     "plugin with directory and ref",
			ref:      "git+https://github.com/jetify-com/devbox-plugins.git?dir=mongodb&ref=my-branch",
			subpath:  "plugin.json",
			expected: "mongodb/plugin.json",
		},
		{
			name:     "plugin with subgroups",
			ref:      "git+https://gitlab.com/group/subgroup/repo.git",
			subpath:  "plugin.json",
			expected: "plugin.json",
		},
		{
			name:     "plugin with SSH URL",
			ref:      "git+ssh://git@github.com/jetify-com/devbox-plugins.git",
			subpath:  "plugin.json",
			expected: "plugin.json",
		},
		{
			name:     "plugin with file URL",
			ref:      "git+file:///tmp/local-repo.git",
			subpath:  "plugin.json",
			expected: "plugin.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := flake.ParseRef(tt.ref)
			if err != nil {
				t.Fatalf("Failed to parse ref %q: %v", tt.ref, err)
			}

			plugin, err := newGitPlugin(ref)
			if err != nil {
				t.Fatalf("Failed to create Git plugin: %v", err)
			}

			// Test that the plugin can be created and the subpath is handled correctly
			// The actual file path will be constructed in FileContent method
			if plugin.ref.Dir != "" {
				expectedPath := plugin.ref.Dir + "/" + tt.subpath
				if expectedPath != tt.expected {
					t.Errorf("Expected path %q, got %q", tt.expected, expectedPath)
				}
			} else {
				if tt.subpath != tt.expected {
					t.Errorf("Expected subpath %q, got %q", tt.expected, tt.subpath)
				}
			}
		})
	}
}

func TestIsBranchName(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected bool
	}{
		{
			name:     "branch name",
			ref:      "main",
			expected: true,
		},
		{
			name:     "branch name with slash",
			ref:      "feature/new-feature",
			expected: true,
		},
		{
			name:     "commit hash",
			ref:      "abc123def456",
			expected: true, // Not 40 chars, so treated as branch
		},
		{
			name:     "full commit hash",
			ref:      "a1b2c3d4e5f6789012345678901234567890abcd",
			expected: false, // 40 chars, looks like commit hash
		},
		{
			name:     "short commit hash",
			ref:      "abc123",
			expected: true, // Not 40 chars, so treated as branch
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBranchName(tt.ref)
			if result != tt.expected {
				t.Errorf("Expected %v for %q, got %v", tt.expected, tt.ref, result)
			}
		})
	}
}
