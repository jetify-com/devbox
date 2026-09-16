package multi

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"go.jetify.com/devbox/internal/devbox/devopt"
	"go.jetify.com/devbox/internal/devconfig/configfile"
)

// TestOpenDeduplicatesConfigsInSameDir ensures that a directory containing more
// than one recognized config name (e.g. both devbox.json and devbox.jsonc) is
// opened only once, so `--all-projects` commands don't run twice for it.
func TestOpenDeduplicatesConfigsInSameDir(t *testing.T) {
	root := t.TempDir()

	// projBoth has both config names; projJSON has only devbox.json.
	projBoth := filepath.Join(root, "projBoth")
	projJSON := filepath.Join(root, "projJSON")
	for _, dir := range []string{projBoth, projJSON} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, configfile.DefaultName), []byte(`{"packages": []}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(projBoth, configfile.AltName), []byte(`{"packages": []}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// multi.Open walks the current working directory.
	t.Chdir(root)

	boxes, err := Open(&devopt.Opts{Stderr: io.Discard})
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	if len(boxes) != 2 {
		t.Errorf("Open() opened %d projects, want 2 (projBoth must be opened once despite having both config names)", len(boxes))
	}
}
