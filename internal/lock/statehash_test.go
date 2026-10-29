package lock

import (
	"os"
	"path/filepath"
	"testing"

	"go.jetify.com/devbox/nix/flake"
)

func TestCompareStateHashes(t *testing.T) {
	saved := stateHashFile{
		ConfigHash:             "config",
		NixConfigHash:          "nix",
		DevboxVersion:          "0.1.0",
		LockFileHash:           "lock",
		NixPrintDevEnvHash:     "printdevenv",
		NixProfileManifestHash: "manifest",
	}

	tests := []struct {
		name   string
		modify func(s *stateHashFile)
		want   StateStatus
	}{
		{"unchanged", func(*stateHashFile) {}, StateUpToDate},
		{"config", func(s *stateHashFile) { s.ConfigHash = "config2" }, StateNonNixStale},
		{"config and nix config", func(s *stateHashFile) {
			s.ConfigHash = "config2"
			s.NixConfigHash = "nix2"
		}, StateStale},
		// Can't happen in practice, but if it does the nix inputs changed.
		{"nix config only", func(s *stateHashFile) { s.NixConfigHash = "nix2" }, StateStale},
		{"config and lockfile", func(s *stateHashFile) {
			s.ConfigHash = "config2"
			s.LockFileHash = "lock2"
		}, StateStale},
		{"config and devbox version", func(s *stateHashFile) {
			s.ConfigHash = "config2"
			s.DevboxVersion = "0.2.0"
		}, StateStale},
		{"config and print-dev-env cache", func(s *stateHashFile) {
			s.ConfigHash = "config2"
			s.NixPrintDevEnvHash = "printdevenv2"
		}, StateStale},
		{"config and profile", func(s *stateHashFile) {
			s.ConfigHash = "config2"
			s.NixProfileManifestHash = "manifest2"
		}, StateStale},
		{"config and shell", func(s *stateHashFile) {
			s.ConfigHash = "config2"
			s.IsFish = true
		}, StateStale},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current := saved
			tt.modify(&current)
			if got := compareStateHashes(&saved, &current); got != tt.want {
				t.Errorf("compareStateHashes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareStateHashesOldStateFile(t *testing.T) {
	// State files written by older versions of Devbox don't have a
	// NixConfigHash. They should trigger one full recompute.
	saved := stateHashFile{ConfigHash: "config"}
	current := stateHashFile{ConfigHash: "config2", NixConfigHash: "nix"}
	if got := compareStateHashes(&saved, &current); got != StateStale {
		t.Errorf("compareStateHashes() = %v, want %v", got, StateStale)
	}
	current.ConfigHash = "config"
	if got := compareStateHashes(&saved, &current); got != StateStale {
		t.Errorf("compareStateHashes() = %v, want %v", got, StateStale)
	}
}

type stateTestProject struct {
	dir           string
	configHash    string
	nixConfigHash string
}

func (p *stateTestProject) ConfigHash() (string, error)    { return p.configHash, nil }
func (p *stateTestProject) NixConfigHash() (string, error) { return p.nixConfigHash, nil }
func (p *stateTestProject) Stdenv() flake.Ref              { return flake.Ref{} }
func (p *stateTestProject) ProjectDir() string             { return p.dir }
func (p *stateTestProject) AllPackageNamesIncludingRemovedTriggerPackages() []string {
	return nil
}

func TestFileStateStatus(t *testing.T) {
	project := &stateTestProject{
		dir:           t.TempDir(),
		configHash:    "config",
		nixConfigHash: "nix",
	}
	if err := os.MkdirAll(filepath.Join(project.dir, ".devbox"), 0o755); err != nil {
		t.Fatal(err)
	}
	lockfile, err := GetFile(project)
	if err != nil {
		t.Fatal(err)
	}
	status := func() StateStatus {
		t.Helper()
		got, err := lockfile.StateStatus(false)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	save := func() {
		t.Helper()
		err := UpdateAndSaveStateHashFile(UpdateStateHashFileArgs{
			ProjectDir:    project.dir,
			ConfigHash:    project.configHash,
			NixConfigHash: project.nixConfigHash,
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	if got := status(); got != StateStale {
		t.Errorf("got %v with no state file, want %v", got, StateStale)
	}

	save()
	if got := status(); got != StateUpToDate {
		t.Errorf("got %v after saving, want %v", got, StateUpToDate)
	}

	project.configHash = "config2"
	if got := status(); got != StateNonNixStale {
		t.Errorf("got %v after changing non-nix inputs, want %v", got, StateNonNixStale)
	}
	if upToDate, err := lockfile.IsUpToDateAndInstalled(false); err != nil || upToDate {
		t.Errorf("IsUpToDateAndInstalled() = %v, %v, want false, nil", upToDate, err)
	}

	save()
	project.configHash = "config3"
	project.nixConfigHash = "nix3"
	if got := status(); got != StateStale {
		t.Errorf("got %v after changing nix inputs, want %v", got, StateStale)
	}

	// A changed print-dev-env cache means the nix environment changed outside
	// of devbox's control, so recompute everything.
	save()
	project.configHash = "config4"
	cache := filepath.Join(project.dir, ".devbox", ".nix-print-dev-env-cache")
	if err := os.WriteFile(cache, []byte(`{"variables": {}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := status(); got != StateStale {
		t.Errorf("got %v after changing print-dev-env cache, want %v", got, StateStale)
	}
}

func TestFileStateStatusOldStateFile(t *testing.T) {
	project := &stateTestProject{
		dir:           t.TempDir(),
		configHash:    "config",
		nixConfigHash: "nix",
	}
	if err := os.MkdirAll(filepath.Join(project.dir, ".devbox"), 0o755); err != nil {
		t.Fatal(err)
	}
	lockfile, err := GetFile(project)
	if err != nil {
		t.Fatal(err)
	}
	// Written by a Devbox version that didn't know about nix_config_hash.
	err = UpdateAndSaveStateHashFile(UpdateStateHashFileArgs{
		ProjectDir: project.dir,
		ConfigHash: "old-config",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := lockfile.StateStatus(false)
	if err != nil {
		t.Fatal(err)
	}
	if got != StateStale {
		t.Errorf("got %v, want %v", got, StateStale)
	}
}
