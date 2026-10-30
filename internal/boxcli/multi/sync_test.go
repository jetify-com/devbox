package multi

import (
	"os"
	"path/filepath"
	"testing"

	"go.jetify.com/devbox/internal/cuecfg"
	"go.jetify.com/devbox/internal/lock"
)

// TestSyncLockfilesOnlyRequestedPackages ensures that when specific packages
// are requested, unversioned or flake entries are not synced along with them.
func TestSyncLockfilesOnlyRequestedPackages(t *testing.T) {
	const (
		oldTime = "2024-01-01T00:00:00Z"
		newTime = "2024-06-01T00:00:00Z"
	)
	keys := []string{"go@latest", "hello", "github:numtide/flake-utils"}
	writeLock := func(dir, lastModified string) string {
		t.Helper()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		f := lock.File{LockFileVersion: "1", Packages: map[string]*lock.Package{}}
		for _, key := range keys {
			f.Packages[key] = &lock.Package{LastModified: lastModified, Resolved: key + "-" + lastModified}
		}
		path := filepath.Join(dir, "devbox.lock")
		if err := cuecfg.WriteFile(path, f); err != nil {
			t.Fatal(err)
		}
		return path
	}

	root := t.TempDir()
	oldLock := writeLock(filepath.Join(root, "old"), oldTime)
	writeLock(filepath.Join(root, "new"), newTime)

	// SyncLockfiles walks the current working directory.
	t.Chdir(root)

	if err := SyncLockfiles([]string{"go"}); err != nil {
		t.Fatalf("SyncLockfiles() error: %v", err)
	}

	var got lock.File
	if err := cuecfg.ParseFile(oldLock, &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"go@latest":                  newTime,
		"hello":                      oldTime,
		"github:numtide/flake-utils": oldTime,
	}
	for key, wantTime := range want {
		if gotTime := got.Packages[key].LastModified; gotTime != wantTime {
			t.Errorf("Packages[%q].LastModified = %q, want %q", key, gotTime, wantTime)
		}
	}
}
