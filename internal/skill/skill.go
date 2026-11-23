// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

// Package skill installs the devbox agent skill (an Agent Skills SKILL.md,
// see https://agentskills.io) so coding agents know to use devbox in projects
// that have a devbox.json.
package skill

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.jetify.com/devbox/internal/build"
)

// Name is the skill's name. Agent Skills requires it to match the name of the
// directory that holds SKILL.md.
const Name = "devbox"

// managedMarker identifies a SKILL.md written by devbox, so that devbox can
// update or remove it without clobbering a file the user wrote themselves.
const managedMarker = "managed-by: devbox"

//go:embed SKILL.md
var skillTemplate string

// Content returns the SKILL.md contents for this version of devbox.
func Content() string {
	return strings.ReplaceAll(skillTemplate, "{{DEVBOX_VERSION}}", build.Version)
}

// Scope is where a skill gets installed.
type Scope string

const (
	// ScopeProject installs into a project so every agent working in that
	// repo picks it up. These files are meant to be committed.
	ScopeProject Scope = "project"
	// ScopeUser installs into the user's home directory so the skill applies
	// to every project on the machine.
	ScopeUser Scope = "user"
)

// Dirs returns the skills directories for a scope. Each agent reads skills
// from its own directory:
//
//   - .agents/skills is the cross-agent location used by Codex, Cursor,
//     GitHub Copilot, Gemini CLI, OpenCode, Amp and others.
//   - .claude/skills is the only location Claude Code reads.
//
// For ScopeProject, root is the project directory. For ScopeUser it's
// ignored.
func Dirs(scope Scope, root string) ([]string, error) {
	switch scope {
	case ScopeProject:
		return []string{
			filepath.Join(root, ".agents", "skills"),
			filepath.Join(root, ".claude", "skills"),
		}, nil
	case ScopeUser:
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		claudeDir := os.Getenv("CLAUDE_CONFIG_DIR")
		if claudeDir == "" {
			claudeDir = filepath.Join(home, ".claude")
		}
		return []string{
			filepath.Join(home, ".agents", "skills"),
			filepath.Join(claudeDir, "skills"),
		}, nil
	default:
		return nil, fmt.Errorf("unknown skill scope %q", scope)
	}
}

// Status of a single SKILL.md after an install or uninstall.
type Status string

const (
	StatusCreated   Status = "created"
	StatusUpdated   Status = "updated"
	StatusUnchanged Status = "unchanged"
	StatusRemoved   Status = "removed"
	StatusNotFound  Status = "not found"
	// StatusSkipped means a SKILL.md that devbox didn't write already exists.
	StatusSkipped Status = "skipped"
)

// Result describes what happened to one SKILL.md.
type Result struct {
	Path   string
	Status Status
}

// Install writes the devbox SKILL.md into each skills directory for scope.
// A SKILL.md that devbox wrote earlier is updated in place. One that devbox
// didn't write is left alone unless force is true.
func Install(scope Scope, root string, force bool) ([]Result, error) {
	dirs, err := Dirs(scope, root)
	if err != nil {
		return nil, err
	}
	content := []byte(Content())
	results := []Result{}
	seen := map[string]bool{}
	for _, dir := range dirs {
		path := filepath.Join(dir, Name, "SKILL.md")
		// Projects often symlink .claude/skills to .agents/skills (this repo
		// does). Don't report the same file twice.
		if key := resolve(path); seen[key] {
			continue
		} else {
			seen[key] = true
		}

		if scope == ScopeProject {
			if err := checkInside(root, path); err != nil {
				return results, err
			}
		}

		status := StatusCreated
		existing, err := readRegularFile(path)
		switch {
		case err == nil && bytes.Equal(existing, content):
			results = append(results, Result{path, StatusUnchanged})
			continue
		case (err == nil && !isManaged(existing) || errors.Is(err, errSymlink)) && !force:
			results = append(results, Result{path, StatusSkipped})
			continue
		case err == nil || errors.Is(err, errSymlink):
			status = StatusUpdated
		case !errors.Is(err, fs.ErrNotExist):
			return results, err
		}

		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return results, err
		}
		if err := replaceFile(path, content); err != nil {
			return results, err
		}
		results = append(results, Result{path, status})
	}
	return results, nil
}

// Uninstall removes the devbox SKILL.md from each skills directory for scope.
// It only removes files devbox wrote, unless force is true.
func Uninstall(scope Scope, root string, force bool) ([]Result, error) {
	dirs, err := Dirs(scope, root)
	if err != nil {
		return nil, err
	}
	results := []Result{}
	seen := map[string]bool{}
	for _, dir := range dirs {
		skillDir := filepath.Join(dir, Name)
		path := filepath.Join(skillDir, "SKILL.md")
		if key := resolve(path); seen[key] {
			continue
		} else {
			seen[key] = true
		}

		if scope == ScopeProject {
			if err := checkInside(root, path); err != nil {
				return results, err
			}
		}
		if _, err := os.Lstat(path); errors.Is(err, fs.ErrNotExist) {
			results = append(results, Result{path, StatusNotFound})
			continue
		}
		existing, err := readRegularFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			results = append(results, Result{path, StatusNotFound})
			continue
		} else if err != nil && !errors.Is(err, errSymlink) {
			return results, err
		}
		if (err != nil || !isManaged(existing)) && !force {
			results = append(results, Result{path, StatusSkipped})
			continue
		}
		if err := os.Remove(path); err != nil {
			return results, err
		}
		// Remove the skill's directory too, but only if it's now empty.
		_ = os.Remove(skillDir)
		results = append(results, Result{path, StatusRemoved})
	}
	return results, nil
}

// IsInstalled reports whether a devbox-managed SKILL.md exists in every
// skills directory for scope.
func IsInstalled(scope Scope, root string) bool {
	dirs, err := Dirs(scope, root)
	if err != nil {
		return false
	}
	for _, dir := range dirs {
		b, err := os.ReadFile(filepath.Join(dir, Name, "SKILL.md"))
		if err != nil || !isManaged(b) {
			return false
		}
	}
	return true
}

// errSymlink is returned by readRegularFile for a symlink. Callers treat a
// symlinked SKILL.md as one devbox didn't write.
var errSymlink = errors.New("SKILL.md is a symlink")

// readRegularFile reads path without following a symlink at path itself, so a
// repository can't point SKILL.md at a file elsewhere on the machine.
func readRegularFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return nil, errSymlink
	}
	return os.ReadFile(path)
}

// replaceFile atomically replaces path with a regular file holding content.
// Renaming over path replaces a symlink itself instead of writing through it.
func replaceFile(path string, content []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".SKILL.md-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) //nolint:errcheck // already renamed on success
	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// checkInside returns an error if path, after resolving symlinked
// directories, isn't inside root. Symlinks between directories within the
// project (such as .claude/skills -> ../.agents/skills) are fine.
func checkInside(root, path string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(realRoot, resolve(absPath))
	if err != nil || !filepath.IsLocal(rel) {
		return fmt.Errorf("refusing to install the skill to %s: it resolves outside the project %s", path, root)
	}
	return nil
}

func isManaged(content []byte) bool {
	return bytes.Contains(content, []byte(managedMarker))
}

// resolve returns path with symlinks in its parent directories evaluated, so
// two paths that name the same file compare equal. It falls back to the
// cleaned path when the directories don't exist yet.
func resolve(path string) string {
	dir := filepath.Dir(path)
	for d := dir; ; d = filepath.Dir(d) {
		if real, err := filepath.EvalSymlinks(d); err == nil {
			rel, _ := filepath.Rel(d, path)
			return filepath.Join(real, rel)
		}
		if parent := filepath.Dir(d); parent == d {
			return filepath.Clean(path)
		}
	}
}
