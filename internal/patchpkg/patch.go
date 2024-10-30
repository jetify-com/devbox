package patchpkg

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os/exec"
	"path"
	"slices"
	"strings"
)

// libPatcher patches ELF binaries to use an alternative version of glibc.
type libPatcher struct {
	// ld is the absolute path to the new dynamic linker (ld.so).
	ld string

	// rpath is the new RPATH with the directories containing the new libc
	// shared objects (libc.so) and other libraries.
	rpath []string

	// needed are shared libraries to add as dependencies (DT_NEEDED).
	needed []string
}

// setGlibc configures the patcher to use the dynamic linker and libc libraries
// in pkg.
func (p *libPatcher) setGlibc(pkg *packageFS) error {
	// Verify that we can find a directory with libc in it.
	glob := "lib*/libc.so*"
	matches, _ := fs.Glob(pkg, glob)
	if len(matches) == 0 {
		return fmt.Errorf("cannot find libc.so file matching %q", glob)
	}
	for i := range matches {
		matches[i] = path.Dir(matches[i])
	}
	// Pick the shortest name: lib < lib32 < lib64 < libx32
	//
	// - lib is usually a symlink to the correct arch (e.g., lib -> lib64)
	// - *.so is usually a symlink to the correct version (e.g., foo.so -> foo.so.2)
	slices.Sort(matches)

	lib, err := pkg.OSPath(matches[0])
	if err != nil {
		return err
	}
	p.rpath = append(p.rpath, lib)
	slog.Debug("found new libc directory", "path", lib)

	// Verify that we can find the new dynamic linker.
	glob = "lib*/ld-linux*.so*"
	matches, _ = fs.Glob(pkg, glob)
	if len(matches) == 0 {
		return fmt.Errorf("cannot find ld.so file matching %q", glob)
	}
	slices.Sort(matches)
	p.ld, err = pkg.OSPath(matches[0])
	if err != nil {
		return err
	}
	slog.Debug("found new dynamic linker", "path", p.ld)
	return nil
}

// setGlibc configures the patcher to use the standard C++ and gcc libraries in
// pkg.
func (p *libPatcher) setGcc(pkg *packageFS) error {
	// Verify that we can find a directory with libstdc++.so in it.
	glob := "lib*/libstdc++.so*"
	matches, _ := fs.Glob(pkg, glob)
	if len(matches) == 0 {
		return fmt.Errorf("cannot find libstdc++.so file matching %q", glob)
	}
	for i := range matches {
		matches[i] = path.Dir(matches[i])
	}
	// Pick the shortest name: lib < lib32 < lib64 < libx32
	//
	// - lib is usually a symlink to the correct arch (e.g., lib -> lib64)
	// - *.so is usually a symlink to the correct version (e.g., foo.so -> foo.so.2)
	slices.Sort(matches)

	lib, err := pkg.OSPath(matches[0])
	if err != nil {
		return err
	}
	p.rpath = append(p.rpath, lib)
	p.needed = append(p.needed, "libstdc++.so")
	slog.Debug("found new libstdc++ directory", "path", lib)
	return nil
}

func (p *libPatcher) prependRPATH(libPkg *packageFS) {
	glob := "lib*/*.so*"
	matches, _ := fs.Glob(libPkg, glob)
	if len(matches) == 0 {
		slog.Debug("not prepending package to RPATH because no shared libraries were found", "pkg", libPkg.storePath)
		return
	}
	for i := range matches {
		matches[i] = path.Dir(matches[i])
	}
	slices.Sort(matches)
	matches = slices.Compact(matches)
	for i := range matches {
		var err error
		matches[i], err = libPkg.OSPath(matches[i])
		if err != nil {
			continue
		}
	}
	p.rpath = append(p.rpath, matches...)
	slog.Debug("prepended package lib dirs to RPATH", "pkg", libPkg.storePath, "dirs", matches)
}

// patch applies glibc patches to a binary and writes the patched result to
// outPath. It does not modify the original binary in-place.
func (p *libPatcher) patch(ctx context.Context, path, outPath string) error {
	cmd := &patchelf{PrintInterpreter: true}
	out, err := cmd.run(ctx, path)
	if err != nil {
		return err
	}
	oldInterp := string(out)

	cmd = &patchelf{PrintRPATH: true}
	out, err = cmd.run(ctx, path)
	if err != nil {
		return err
	}
	oldRpath := strings.Split(string(out), ":")

	cmd = &patchelf{
		SetInterpreter: p.ld,
		SetRPATH:       append(p.rpath, oldRpath...),
		AddNeeded:      p.needed,
		Output:         outPath,
	}
	slog.Debug("patching glibc on binary",
		"path", path, "outPath", cmd.Output,
		"old_interp", oldInterp, "new_interp", cmd.SetInterpreter,
		"old_rpath", oldRpath, "new_rpath", cmd.SetRPATH,
	)
	_, err = cmd.run(ctx, path)
	return err
}

// patchelf runs the patchelf command.
type patchelf struct {
	SetRPATH   []string
	PrintRPATH bool

	SetInterpreter   string
	PrintInterpreter bool

	AddNeeded []string

	Output string
}

// run runs patchelf on an ELF binary and returns its output.
func (p *patchelf) run(ctx context.Context, elf string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, lookPath("patchelf"))
	if len(p.SetRPATH) != 0 {
		cmd.Args = append(cmd.Args, "--force-rpath", "--set-rpath", strings.Join(p.SetRPATH, ":"))
	}
	if p.PrintRPATH {
		cmd.Args = append(cmd.Args, "--print-rpath")
	}
	if p.SetInterpreter != "" {
		cmd.Args = append(cmd.Args, "--set-interpreter", p.SetInterpreter)
	}
	if p.PrintInterpreter {
		cmd.Args = append(cmd.Args, "--print-interpreter")
	}
	for _, needed := range p.AddNeeded {
		cmd.Args = append(cmd.Args, "--add-needed", needed)
	}
	if p.Output != "" {
		cmd.Args = append(cmd.Args, "--output", p.Output)
	}
	cmd.Args = append(cmd.Args, elf)
	out, err := cmd.Output()
	return bytes.TrimSpace(out), err
}
