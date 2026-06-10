// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

// Package flakegen scaffolds a small wrapper flake.nix next to an existing
// Nix expression file so that a directory of Nix expressions can be consumed
// as a local flake (e.g. via a devbox.json `"./pkg": ""` package entry).
package flakegen

import (
	"bytes"
	_ "embed"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pkg/errors"

	"go.jetify.com/devbox/internal/boxcli/usererr"
)

// DefaultNixpkgsURL is the nixpkgs input pinned in generated flakes when the
// caller does not supply one.
const DefaultNixpkgsURL = "github:NixOS/nixpkgs/nixpkgs-unstable"

// defaultNixFile is the file name assumed when the caller points at a
// directory rather than a specific .nix file.
const defaultNixFile = "default.nix"

//go:embed flake-wrapper.nix.tmpl
var wrapperTmplString string

var wrapperTmpl = template.Must(
	template.New("flake-wrapper").Parse(wrapperTmplString),
)

// Opts controls how Generate scaffolds a wrapper flake.
type Opts struct {
	// NixFile is the path to the Nix expression file that the generated
	// flake should callPackage. The flake.nix is written next to it (i.e.
	// filepath.Dir(NixFile)). It is expected to have already been resolved
	// via ResolveNixFile.
	NixFile string

	// NixpkgsURL is the nixpkgs input URL pinned in the generated flake. If
	// empty, DefaultNixpkgsURL is used.
	NixpkgsURL string

	// Attr is the attribute name exposed under `packages.${system}`. If
	// empty, "default" is used.
	Attr string

	// Force overwrites an existing flake.nix when true.
	Force bool

	// Print causes the rendered template to be written to Out instead of a
	// flake.nix file.
	Print bool

	// Out receives either the rendered template (when Print is true) or is
	// unused otherwise. Generate does not print any user-facing summary;
	// callers are responsible for that.
	Out io.Writer
}

// ResolveNixFile turns a user-supplied path into the absolute path of the
// Nix expression file that Generate should wrap.
//
// target can be a directory with a default.nix file, or a specific .nix file.
func ResolveNixFile(target string) (string, error) {
	abs, err := filepath.Abs(target)
	if err != nil {
		return "", errors.WithStack(err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", usererr.New("%s does not exist", abs)
		}
		return "", errors.WithStack(err)
	}
	if info.Mode().IsRegular() {
		if !strings.HasSuffix(filepath.Base(abs), ".nix") {
			return "", usererr.New(
				"%s is a file but does not have a .nix extension. "+
					"Pass a .nix file or a directory containing a "+
					"default.nix.",
				abs,
			)
		}
		return abs, nil
	}
	if !info.IsDir() {
		return "", usererr.New("%s is not a regular file or directory", abs)
	}
	nixPath := filepath.Join(abs, defaultNixFile)
	if _, err := os.Stat(nixPath); err != nil {
		if os.IsNotExist(err) {
			return "", usererr.New(
				"no %s found in %s. Pass a directory containing a "+
					"%s, or point directly at a .nix file.",
				defaultNixFile, abs, defaultNixFile,
			)
		}
		return "", errors.WithStack(err)
	}
	return nixPath, nil
}

// Generate renders the wrapper template for the Nix file described by opts.
// When opts.Print is true, the rendered flake is written to opts.Out and the
// returned path is empty. Otherwise, a flake.nix file is written into the
// directory containing opts.NixFile and its absolute path is returned.
func Generate(opts Opts) (string, error) {
	if opts.NixFile == "" {
		return "", errors.New("flakegen: Opts.NixFile is required")
	}
	nixPath, err := filepath.Abs(opts.NixFile)
	if err != nil {
		return "", errors.WithStack(err)
	}
	nixBase := filepath.Base(nixPath)
	if !strings.HasSuffix(nixBase, ".nix") {
		return "", usererr.New(
			"flakegen: NixFile %q must have a .nix extension", opts.NixFile,
		)
	}
	dir := filepath.Dir(nixPath)
	if _, err := os.Stat(nixPath); err != nil {
		if os.IsNotExist(err) {
			return "", usererr.New(
				"no %s found in %s. "+
					"flakegen expects the file to exist next to the "+
					"generated flake.nix.",
				nixBase, dir,
			)
		}
		return "", errors.WithStack(err)
	}

	nixpkgsURL := opts.NixpkgsURL
	if nixpkgsURL == "" {
		nixpkgsURL = DefaultNixpkgsURL
	}
	attr := opts.Attr
	if attr == "" {
		attr = "default"
	}

	var buf bytes.Buffer
	if err := wrapperTmpl.Execute(&buf, struct {
		Description string
		NixpkgsURL  string
		Attr        string
		NixFile     string
	}{
		Description: "devbox wrapper flake for " + filepath.Base(dir),
		NixpkgsURL:  nixpkgsURL,
		Attr:        attr,
		NixFile:     nixBase,
	}); err != nil {
		return "", errors.WithStack(err)
	}

	if opts.Print {
		if opts.Out == nil {
			return "", errors.New("flakegen: Opts.Out is required when Print is true")
		}
		if _, err := opts.Out.Write(buf.Bytes()); err != nil {
			return "", errors.WithStack(err)
		}
		return "", nil
	}

	flakePath := filepath.Join(dir, "flake.nix")
	if _, err := os.Stat(flakePath); err == nil && !opts.Force {
		return "", usererr.New(
			"%s already exists. Re-run with --force to overwrite.",
			flakePath,
		)
	} else if err != nil && !os.IsNotExist(err) {
		return "", errors.WithStack(err)
	}

	if err := os.WriteFile(flakePath, buf.Bytes(), 0o644); err != nil {
		return "", errors.WithStack(err)
	}
	return flakePath, nil
}
