package patchpkg

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// writeELF writes a minimal 64-bit little-endian ELF executable with the given
// program header types.
func writeELF(t *testing.T, path string, progTypes ...elf.ProgType) {
	t.Helper()

	buf := &bytes.Buffer{}
	hdr := elf.Header64{
		Type:      uint16(elf.ET_EXEC),
		Machine:   uint16(elf.EM_X86_64),
		Version:   uint32(elf.EV_CURRENT),
		Phoff:     64,
		Ehsize:    64,
		Phentsize: 56,
		Phnum:     uint16(len(progTypes)),
		Shentsize: 64,
	}
	copy(hdr.Ident[:], elf.ELFMAG)
	hdr.Ident[elf.EI_CLASS] = byte(elf.ELFCLASS64)
	hdr.Ident[elf.EI_DATA] = byte(elf.ELFDATA2LSB)
	hdr.Ident[elf.EI_VERSION] = byte(elf.EV_CURRENT)
	if err := binary.Write(buf, binary.LittleEndian, hdr); err != nil {
		t.Fatal(err)
	}
	for _, typ := range progTypes {
		prog := elf.Prog64{Type: uint32(typ)}
		if err := binary.Write(buf, binary.LittleEndian, prog); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestHasInterpreter(t *testing.T) {
	dir := t.TempDir()
	writeELF(t, filepath.Join(dir, "dynamic"), elf.PT_PHDR, elf.PT_INTERP, elf.PT_LOAD)
	writeELF(t, filepath.Join(dir, "static"), elf.PT_LOAD)
	if err := os.WriteFile(filepath.Join(dir, "script"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	pkg := newPackageFS(dir)
	tests := map[string]bool{"dynamic": true, "static": false, "script": false}
	for name, want := range tests {
		f, err := pkg.Open(name)
		if err != nil {
			t.Fatal(err)
		}
		if got := hasInterpreter(f); got != want {
			t.Errorf("hasInterpreter(%s) = %t, want %t", name, got, want)
		}
		f.Close()
	}
}

// TestCopyFileStaticBinary checks that statically linked binaries are copied
// as-is instead of being passed to patchelf, which fails on them.
func TestCopyFileStaticBinary(t *testing.T) {
	src := t.TempDir()
	writeELF(t, filepath.Join(src, "bin", "static"), elf.PT_LOAD)
	out := t.TempDir()
	if err := os.Mkdir(filepath.Join(out, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Make sure the test fails if it tries to run patchelf.
	t.Setenv("patchelf", filepath.Join(t.TempDir(), "missing"))
	d := &DerivationBuilder{Glibc: "/nix/store/glibc", glibcPatcher: &libPatcher{}}
	err := d.copyFile(t.Context(), newPackageFS(src), newPackageFS(out), "bin/static")
	if err != nil {
		t.Fatalf("copyFile returned error for static binary: %v", err)
	}

	want, err := os.ReadFile(filepath.Join(src, "bin", "static"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(out, "bin", "static"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Error("copied static binary doesn't match the original")
	}
}
