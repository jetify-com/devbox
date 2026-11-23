// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package skill

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestContentFrontmatter(t *testing.T) {
	content := Content()
	if strings.Contains(content, "{{DEVBOX_VERSION}}") {
		t.Error("version placeholder was not replaced")
	}
	parts := strings.SplitN(content, "---\n", 3)
	if len(parts) != 3 || parts[0] != "" {
		t.Fatal("SKILL.md must start with YAML frontmatter")
	}
	fm := parts[1]
	if !regexp.MustCompile(`(?m)^name: ` + Name + `$`).MatchString(fm) {
		t.Errorf("frontmatter name must be %q", Name)
	}
	m := regexp.MustCompile(`(?m)^description: (.+)$`).FindStringSubmatch(fm)
	if m == nil || len(m[1]) > 1024 {
		t.Error("frontmatter needs a description of at most 1024 characters")
	}
	if !strings.Contains(fm, managedMarker) {
		t.Error("frontmatter is missing the managed-by marker")
	}
	// Agent Skills recommends keeping SKILL.md under 500 lines. Ours should
	// stay much leaner than that.
	if n := strings.Count(content, "\n"); n > 200 {
		t.Errorf("SKILL.md has %d lines, keep it under 200", n)
	}
}

func TestInstallUninstallProject(t *testing.T) {
	root := t.TempDir()
	results, err := Install(ScopeProject, root, false)
	if err != nil {
		t.Fatal(err)
	}
	assertStatuses(t, results, StatusCreated, StatusCreated)
	for _, rel := range []string{".agents/skills/devbox/SKILL.md", ".claude/skills/devbox/SKILL.md"} {
		b, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != Content() {
			t.Errorf("%s has unexpected contents", rel)
		}
	}
	if !IsInstalled(ScopeProject, root) {
		t.Error("IsInstalled = false after Install")
	}

	results, err = Install(ScopeProject, root, false)
	if err != nil {
		t.Fatal(err)
	}
	assertStatuses(t, results, StatusUnchanged, StatusUnchanged)

	// An older devbox-managed version gets updated.
	old := filepath.Join(root, ".claude/skills/devbox/SKILL.md")
	writeFile(t, old, "---\nname: devbox\nmetadata:\n  managed-by: devbox\n---\nold\n")
	results, err = Install(ScopeProject, root, false)
	if err != nil {
		t.Fatal(err)
	}
	assertStatuses(t, results, StatusUnchanged, StatusUpdated)

	results, err = Uninstall(ScopeProject, root, false)
	if err != nil {
		t.Fatal(err)
	}
	assertStatuses(t, results, StatusRemoved, StatusRemoved)
	if _, err := os.Stat(filepath.Join(root, ".agents/skills/devbox")); !os.IsNotExist(err) {
		t.Error("empty skill directory was not removed")
	}
	if IsInstalled(ScopeProject, root) {
		t.Error("IsInstalled = true after Uninstall")
	}
}

func TestInstallKeepsUserSkill(t *testing.T) {
	root := t.TempDir()
	mine := filepath.Join(root, ".agents/skills/devbox/SKILL.md")
	writeFile(t, mine, "---\nname: devbox\n---\nmy own skill\n")

	results, err := Install(ScopeProject, root, false)
	if err != nil {
		t.Fatal(err)
	}
	assertStatuses(t, results, StatusSkipped, StatusCreated)

	results, err = Uninstall(ScopeProject, root, false)
	if err != nil {
		t.Fatal(err)
	}
	assertStatuses(t, results, StatusSkipped, StatusRemoved)
	if _, err := os.Stat(mine); err != nil {
		t.Error("uninstall removed a SKILL.md that devbox didn't write")
	}

	results, err = Install(ScopeProject, root, true)
	if err != nil {
		t.Fatal(err)
	}
	assertStatuses(t, results, StatusUpdated, StatusCreated)
}

func TestInstallSymlinkedSkillsDir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".agents/skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../.agents/skills", filepath.Join(root, ".claude/skills")); err != nil {
		t.Skip("symlinks not supported:", err)
	}
	results, err := Install(ScopeProject, root, false)
	if err != nil {
		t.Fatal(err)
	}
	assertStatuses(t, results, StatusCreated)
}

func TestUserScopeHonorsClaudeConfigDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, "claude-config"))
	dirs, err := Dirs(ScopeUser, "")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join(home, ".agents", "skills"),
		filepath.Join(home, "claude-config", "skills"),
	}
	if strings.Join(dirs, ",") != strings.Join(want, ",") {
		t.Errorf("Dirs(ScopeUser) = %v, want %v", dirs, want)
	}
}

func assertStatuses(t *testing.T, results []Result, want ...Status) {
	t.Helper()
	got := make([]Status, len(results))
	for i, r := range results {
		got[i] = r.Status
	}
	if len(got) != len(want) {
		t.Fatalf("got statuses %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got statuses %v, want %v", got, want)
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestInstallDoesNotWriteThroughSymlinks(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "bashrc")
	writeFile(t, outside, "export SECRET=1\n")

	link := filepath.Join(root, ".agents/skills/devbox/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Skip("symlinks not supported:", err)
	}

	results, err := Install(ScopeProject, root, false)
	if err != nil {
		t.Fatal(err)
	}
	assertStatuses(t, results, StatusSkipped, StatusCreated)

	// --force replaces the link with a regular file and leaves the target alone.
	if _, err := Install(ScopeProject, root, true); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(outside); string(b) != "export SECRET=1\n" {
		t.Errorf("install wrote through a symlink to %s", outside)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink != 0 {
		t.Errorf("SKILL.md symlink was not replaced with a regular file")
	}

	// A dangling link is not followed either.
	dangling := filepath.Join(t.TempDir(), "created-by-devbox")
	other := filepath.Join(root, ".claude/skills/devbox/SKILL.md")
	if err := os.Remove(other); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(dangling, other); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(ScopeProject, root, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dangling); !os.IsNotExist(err) {
		t.Errorf("install created %s through a dangling symlink", dangling)
	}
}

func TestInstallRejectsSkillDirOutsideProject(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, ".agents/skills")); err != nil {
		t.Skip("symlinks not supported:", err)
	}
	if _, err := Install(ScopeProject, root, true); err == nil {
		t.Fatal("Install succeeded for a skills directory outside the project")
	}
	if _, err := os.Stat(filepath.Join(outside, "devbox")); !os.IsNotExist(err) {
		t.Error("install created files outside the project")
	}
}
