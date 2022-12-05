package openssh

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
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
		"Match":              false,
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

func TestSetupDevbox(t *testing.T) {
	testdirs := []string{
		"testdata/no-config",
		"testdata/existing-config",
		"testdata/already-setup",
	}
	for _, testdata := range testdirs {
		t.Run(testdata, func(t *testing.T) {
			workdir := duplicateDir(t, filepath.Join(testdata, "in"))
			t.Setenv("HOME", workdir)

			if err := SetupDevbox(); err != nil {
				t.Error("got SetupDevbox() error:", err)
			}
			dirsEqual(t, workdir, filepath.Join(testdata, "out"))
		})
	}
}

func TestAddVMKey(t *testing.T) {
	workdir := duplicateDir(t, "testdata/add-key/in")
	t.Setenv("HOME", workdir)

	// Must match the filename and content of
	// testdata/add-key/out/.config/devbox/ssh/keys/test.vm.devbox-vms.internal
	host := "test.vm.devbox-vms.internal"
	key := `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACAFBvrkKm1Zrwbpp7DAmTWj7i+S+cjGhAwm++fELg1OpwAAAKjD4XIZw+Fy
GQAAAAtzc2gtZWQyNTUxOQAAACAFBvrkKm1Zrwbpp7DAmTWj7i+S+cjGhAwm++fELg1Opw
AAAEAcvFwROtvcGVsdSg73Y+znyO9F6LFRxhWa7UJdGcjGzwUG+uQqbVmvBumnsMCZNaPu
L5L5yMaEDCb758QuDU6nAAAAH2djdXJ0aXNAR3JlZ3MtTWFjQm9vay1Qcm8ubG9jYWwBAg
MEBQY=
-----END OPENSSH PRIVATE KEY-----
`
	if err := AddVMKey(host, key); err != nil {
		t.Error("got AddKey(host, key) error:", err)
	}
	dirsEqual(t, workdir, "testdata/add-key/out")
}

func duplicateDir(t *testing.T, dir string) string {
	srcFS := os.DirFS(dir)
	dstDir := t.TempDir()
	err := fs.WalkDir(srcFS, ".", func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			// Just return an empty temp dir if the directory
			// doesn't exist.
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if path == "." {
			return nil
		}

		info, err := fs.Stat(srcFS, path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return os.Mkdir(filepath.Join(dstDir, path), info.Mode().Perm())
		}
		data, err := fs.ReadFile(srcFS, path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dstDir, path), data, info.Mode().Perm())
	})
	if err != nil {
		t.Fatal("got error duplicating testdata dir:", err)
	}
	return dstDir
}

func dirsEqual(t *testing.T, gotDir, wantDir string) {
	var perms map[string]fs.FileMode
	if b, err := os.ReadFile(filepath.Join(wantDir, "perms.json")); err == nil {
		v := make(map[string]string)
		if err := json.Unmarshal(b, &v); err == nil {
			perms = make(map[string]fs.FileMode, len(v))
			for path, modeStr := range v {
				mode, err := strconv.ParseUint(modeStr, 8, 32)
				if err != nil {
					t.Fatalf("path %q has invalid permissions %q", path, modeStr)
				}
				perms[path] = fs.FileMode(mode)
			}
		}
	}
	wantFS := os.DirFS(wantDir)
	gotFS := os.DirFS(gotDir)
	err := fs.WalkDir(wantFS, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			// If the wantDir is missing then there's nothing to
			// check.
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			t.Error("got WalkDir error:", err)
		}

		// perms.json is a special-case where we put expected file
		// permissions.
		if path == "." || path == "perms.json" {
			return nil
		}

		if wantPerm, ok := perms[path]; ok {
			gotInfo, err := fs.Stat(gotFS, path)
			if err != nil {
				t.Fatalf("Stat(got, %q): %v", path, err)
			}
			if got, want := gotInfo.Mode().Perm(), wantPerm; got != want {
				t.Errorf("wrong permissions at %q: got %q, want %q", path, got, want)
			}
		}
		if entry.IsDir() {
			return nil
		}

		gotBytes, err := fs.ReadFile(gotFS, path)
		if err != nil {
			t.Errorf("ReadFile(got, %q): %v", path, err)
		}
		wantBytes, err := fs.ReadFile(wantFS, path)
		if err != nil {
			t.Errorf("ReadFile(want, %q): %v", path, err)
		}

		if diff := cmp.Diff(os.ExpandEnv(string(wantBytes)), os.ExpandEnv(string(gotBytes))); diff != "" {
			t.Errorf("wrong file contents at %q (-want +got):\n%s", path, diff)
		}
		return nil
	})
	if err != nil {
		t.Fatal("got error checking if dirs are equal:", err)
	}
}
