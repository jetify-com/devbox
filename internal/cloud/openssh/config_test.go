// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package openssh

import (
	"bytes"
	_ "embed"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"

	"go.jetpack.io/devbox/internal/envir"
)

func TestDevboxIncludeRegex(t *testing.T) {
	tests := map[string]bool{
		`Include ~/.config/devbox/ssh/config`:       true,
		`include ~/.config/devbox/ssh/config`:       true,
		`Include "~/.config/devbox/ssh/config"`:     true,
		`"Include" ~/.config/devbox/ssh/config`:     true,
		`"Include" "~/.config/devbox/ssh/config"`:   true,
		`"Include" = "~/.config/devbox/ssh/config"`: true,
		`Include = "~/.config/devbox/ssh/config"`:   true,
		`Include=~/.config/devbox/ssh/config`:       true,
		"Include\t~/.config/devbox/ssh/config":      true,
		"\tInclude ~/.config/devbox/ssh/config":     true,
		` Include ~/.config/devbox/ssh/config`:      true,

		`# Include ~/.config/devbox/ssh/config`:                false,
		`Include ~/.config/blah # ~/.config/devbox/ssh/config`: false,
		`Include`:                            false,
		`Include ~/.config/blah`:             false,
		`Include~/.config/devbox/ssh/config`: false,
		`IdentityFile ~/.config/devbox/ssh/config/keys/mykey`: false,
		`Hostname include devbox/ssh/config`:                  false,
	}
	for in, match := range tests {
		t.Run(in, func(t *testing.T) {
			got := reDevboxInclude.MatchString(in)
			if got != match {
				t.Errorf("got wrong match for %q\ngot match = %t, want %t", in, got, match)
			}
		})
	}
}

func TestHostOrMatchRegex(t *testing.T) {
	tests := map[string]bool{
		`Host *.devbox.sh`:   true,
		`  Host *.devbox.sh`: true,
		"\tHost *.devbox.sh": true,
		`Match all`:          true,
		`  Match all`:        true,
		"\tMatch all":        true,

		"Host":               false,
		"Hostname devbox.sh": false,
		`# Host *.devbox.sh`: true,
		`# Match all`:        true,
	}
	for in, match := range tests {
		t.Run(in, func(t *testing.T) {
			got := reHostOrMatch.MatchString(in)
			if got != match {
				t.Errorf("got wrong match for %q\ngot match = %t, want %t", in, got, match)
			}
		})
	}
}

//go:embed testdata/devbox-ssh-config.golden
var goldenDevboxSSHConfig []byte

func TestSetupDevbox(t *testing.T) {
	want := fstest.MapFS{
		".config":            &fstest.MapFile{Mode: fs.ModeDir | 0o755},
		".config/devbox":     &fstest.MapFile{Mode: fs.ModeDir | 0o755},
		".config/devbox/ssh": &fstest.MapFile{Mode: fs.ModeDir | 0o700},
		".config/devbox/ssh/config": &fstest.MapFile{
			Data: goldenDevboxSSHConfig,
			Mode: 0o644,
		},
		".config/devbox/ssh/known_hosts": &fstest.MapFile{
			Data: sshKnownHosts,
			Mode: 0o644,
		},
		".config/devbox/ssh/sockets": &fstest.MapFile{Mode: fs.ModeDir | 0o700},

		".ssh": &fstest.MapFile{Mode: fs.ModeDir | 0o700},
		".ssh/config": &fstest.MapFile{
			Data: []byte("Include \"$HOME/.config/devbox/ssh/config\"\n"),
			Mode: 0o644,
		},
	}

	t.Run("NoConfigs", func(t *testing.T) {
		in := fstest.MapFS{}
		workdir := fsToDir(t, in)
		t.Setenv(envir.Home, workdir)
		if err := SetupDevbox(); err != nil {
			t.Error("got SetupDevbox() error:", err)
		}
		got := os.DirFS(workdir)
		fsEqual(t, got, want)
	})
	t.Run("ExistingSSHConfig", func(t *testing.T) {
		existingSSHConfig := []byte("Host example.com\n\tUser example\n\tPort 1234\n")
		input := fstest.MapFS{
			".ssh": &fstest.MapFile{Mode: fs.ModeDir | 0o700},
			".ssh/config": &fstest.MapFile{
				Data: existingSSHConfig,
				Mode: 0o644,
			},
		}
		// Temporarily change the desired ~/.ssh/config so it contains
		// the initial contents of the input ~/.ssh/config.
		originalWantConfig := want[".ssh/config"]
		defer func() { want[".ssh/config"] = originalWantConfig }()
		want[".ssh/config"] = &fstest.MapFile{
			Data: append(
				// fsEqual will expand $HOME so this becomes an absolute path.
				[]byte("Include \"$HOME/.config/devbox/ssh/config\"\n"),
				existingSSHConfig...,
			),
			Mode: 0o644,
		}

		workdir := fsToDir(t, input)
		t.Setenv(envir.Home, workdir)
		if err := SetupDevbox(); err != nil {
			t.Error("got SetupDevbox() error:", err)
		}
		got := os.DirFS(workdir)
		fsEqual(t, got, want)
	})
	t.Run("AlreadySetup", func(t *testing.T) {
		in := want
		workdir := fsToDir(t, in)
		t.Setenv(envir.Home, workdir)
		if err := SetupDevbox(); err != nil {
			t.Error("got SetupDevbox() error:", err)
		}
		got := os.DirFS(workdir)
		fsEqual(t, got, want)
	})
}

//go:embed testdata/devbox-ssh-debug-config.golden
var goldenDevboxSSHDebugConfig []byte

func TestSetupInsecureDebug(t *testing.T) {
	wantAddr := "127.0.0.1:2222"
	want := fstest.MapFS{
		".config":            &fstest.MapFile{Mode: fs.ModeDir | 0o755},
		".config/devbox":     &fstest.MapFile{Mode: fs.ModeDir | 0o755},
		".config/devbox/ssh": &fstest.MapFile{Mode: fs.ModeDir | 0o700},
		".config/devbox/ssh/config": &fstest.MapFile{
			Data: goldenDevboxSSHDebugConfig,
			Mode: 0o644,
		},
		".config/devbox/ssh/known_hosts": &fstest.MapFile{
			Data: sshKnownHosts,
			Mode: 0o644,
		},
		".config/devbox/ssh/known_hosts_debug": &fstest.MapFile{Mode: 0o644},
		".config/devbox/ssh/sockets":           &fstest.MapFile{Mode: fs.ModeDir | 0o700},

		".ssh": &fstest.MapFile{Mode: fs.ModeDir | 0o700},
		".ssh/config": &fstest.MapFile{
			Data: []byte("Include \"$HOME/.config/devbox/ssh/config\"\n"),
			Mode: 0o644,
		},
	}

	t.Run("NoConfigs", func(t *testing.T) {
		in := fstest.MapFS{}
		workdir := fsToDir(t, in)
		t.Setenv(envir.Home, workdir)
		if err := SetupInsecureDebug(wantAddr); err != nil {
			t.Errorf("got SetupInsecureDebug(%q) error: %v", wantAddr, err)
		}
		got := os.DirFS(workdir)
		fsEqual(t, got, want)
	})
	t.Run("ChangeHost", func(t *testing.T) {
		input := fstest.MapFS{}
		for k, v := range want {
			input[k] = v
		}

		// Set the initial config to have a debug host = 127.0.0.2 so we
		// can check that it gets changed back to 127.0.0.1.
		input[".config/devbox/ssh/config"] = &fstest.MapFile{
			Data: bytes.ReplaceAll(goldenDevboxSSHDebugConfig, []byte("127.0.0.1"), []byte("127.0.0.2")),
			Mode: 0o644,
		}

		// Put something in known_hosts_debug so we can check that it
		// gets cleared out.
		input[".config/devbox/ssh/known_hosts_debug"] = &fstest.MapFile{
			Data: []byte("[127.0.0.1]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAPY1ms2jt+QPvhq89J8KF7rfTCFUi6X6Ik4O9EIAT/c\n"),
			Mode: 0o644,
		}

		workdir := fsToDir(t, input)
		t.Setenv(envir.Home, workdir)
		if err := SetupInsecureDebug(wantAddr); err != nil {
			t.Errorf("got SetupInsecureDebug(%q) error: %v", wantAddr, err)
		}
		got := os.DirFS(workdir)
		fsEqual(t, got, want)
	})
}

//go:embed testdata/test.vm.devbox-vms.internal.golden
var goldenVMKey []byte

func TestAddVMKey(t *testing.T) {
	host := "test.vm.devbox-vms.internal"
	input := fstest.MapFS{}
	want := fstest.MapFS{
		".config":                 &fstest.MapFile{Mode: fs.ModeDir | 0o755},
		".config/devbox":          &fstest.MapFile{Mode: fs.ModeDir | 0o755},
		".config/devbox/ssh":      &fstest.MapFile{Mode: fs.ModeDir | 0o700},
		".config/devbox/ssh/keys": &fstest.MapFile{Mode: fs.ModeDir | 0o700},

		".config/devbox/ssh/keys/" + host: &fstest.MapFile{
			Data: goldenVMKey,
			Mode: 0o600,
		},
	}

	workdir := fsToDir(t, input)
	t.Setenv(envir.Home, workdir)
	if err := AddVMKey(host, string(goldenVMKey)); err != nil {
		t.Error("got AddKey(host, key) error:", err)
	}
	got := os.DirFS(workdir)
	fsEqual(t, got, want)
}

// fsEqual checks if the contents of two file systems are the same. Two file
// systems are equal if their path hierarchies are the same and the file at
// each path passes fsPathsEqual. It ignores the mode of the root directory of
// each file system.
//
// fsEqual will report as many equality errors as possible by continuing to walk
// the tree after a file comparison fails.
func fsEqual(t *testing.T, got, want fs.FS) {
	t.Helper()

	checked := map[string]bool{}
	err := fs.WalkDir(got, ".", func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}
		checked[path] = true
		fsPathsEqual(t, got, want, path)
		return nil
	})
	if err != nil {
		t.Fatal("got error checking if file systems are equal:", err)
	}

	err = fs.WalkDir(want, ".", func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." || checked[path] {
			return nil
		}
		fsPathsEqual(t, got, want, path)
		return nil
	})
	if err != nil {
		t.Fatal("got error checking if file systems are equal:", err)
	}
}

// fsPathsEqual checks if the file at a path in two file systems is the same.
// Two file paths are equal if:
//
//   - their file contents are the same after calling os.ExpandEnv on each one.
//   - their file mode dir bits are the same.
//   - their file permission mode bits are the same.
//
// It does not consider any other file info such as ModTime or other file mode
// bits.
//
// Expanding environment variables in files makes it easier for tests to define
// golden files with dynamic content. For example, a test can create an
// fstest.MapFile with the data {"name": "$USER"} and compare it to some test
// results to make sure there's a JSON file containing the current username.
func fsPathsEqual(t *testing.T, gotFS, wantFS fs.FS, path string) {
	t.Helper()

	gotInfo, err := fs.Stat(gotFS, path)
	if errors.Is(err, fs.ErrNotExist) {
		t.Errorf("got a missing file at %q", path)
		return
	}
	if err != nil {
		t.Errorf("got fs.Stat(gotFS, %q) error: %v", path, err)
	}
	wantInfo, err := fs.Stat(wantFS, path)
	if errors.Is(err, fs.ErrNotExist) {
		t.Errorf("got an extra file at %q", path)
		return
	}
	if err != nil {
		t.Errorf("got fs.Stat(wantFS, %q) error: %v", path, err)
	}

	// Bail early to avoid nil pointer panics.
	if gotInfo == nil || wantInfo == nil {
		return
	}
	if got, want := gotInfo.Mode().Perm(), wantInfo.Mode().Perm(); got != want {
		t.Errorf("got %q permissions %s, want %s", path, got, want)
	}
	if gotInfo.IsDir() != wantInfo.IsDir() {
		gotType, wantType := "file", "file"
		if gotInfo.IsDir() {
			gotType = "directory"
		}
		if wantInfo.IsDir() {
			wantType = "directory"
		}
		t.Errorf("got a %s at path %q, want a %s", gotType, path, wantType)
	}

	// No need to compare file contents if either path is a directory.
	if gotInfo.IsDir() || wantInfo.IsDir() {
		return
	}

	gotBytes, err := fs.ReadFile(gotFS, path)
	if err != nil {
		t.Errorf("got fs.ReadFile(gotFS, %q) error: %v", path, err)
	}
	wantBytes, err := fs.ReadFile(wantFS, path)
	if err != nil {
		t.Errorf("got fs.ReadFile(wantFS, %q) error: %v", path, err)
	}
	diff := cmp.Diff(os.ExpandEnv(string(wantBytes)), os.ExpandEnv(string(gotBytes)))
	if diff != "" {
		t.Errorf("got wrong file contents at %q (-want +got):\n%s", path, diff)
	}
}

// fsToDir writes a file system to a local temp directory. It replicates each
// file's contents and permissions, but ignores any other file info. If the
// root of the file system returns [fs.ErrNotExist], then fsToDir returns an
// empty temp directory.
func fsToDir(t *testing.T, fsys fs.FS) (dir string) {
	t.Helper()

	dir = t.TempDir()
	err := fs.WalkDir(fsys, ".", func(path string, entry fs.DirEntry, err error) error {
		if path == "." {
			// Just return an empty directory if the input is also
			// empty.
			if err == nil || errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}

		info, err := fs.Stat(fsys, path)
		if err != nil {
			t.Error("got error writing fs to dir:", err)
			return nil
		}
		if info.Mode().Perm() == 0 {
			// It's impossible to create a file that you can't write
			// to.
			t.Fatalf("got error writing fs to dir: path %q has empty permissions", path)
		}
		if entry.IsDir() {
			if err := os.Mkdir(filepath.Join(dir, path), info.Mode().Perm()); err != nil {
				t.Error("got error writing fs to dir:", err)
			}
			return nil
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			t.Error("got error writing fs to dir:", err)
			return nil
		}
		if err := os.WriteFile(filepath.Join(dir, path), data, info.Mode().Perm()); err != nil {
			t.Error("got error writing fs to dir:", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal("got error writing fs to dir:", err)
	}
	return dir
}
