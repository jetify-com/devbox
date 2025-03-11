package nix

import (
	"go.jetify.com/devbox/nix"
)

// The types and functions in this file act a shim for the non-internal version
// of this package (go.jetify.com/devbox/nix). That way callers don't need to
// import two nix packages and alias one of them.

const (
	Version2_12 = nix.Version2_12
	Version2_13 = nix.Version2_13
	Version2_14 = nix.Version2_14
	Version2_15 = nix.Version2_15
	Version2_16 = nix.Version2_16
	Version2_17 = nix.Version2_17
	Version2_18 = nix.Version2_18
	Version2_19 = nix.Version2_19
	Version2_20 = nix.Version2_20
	Version2_21 = nix.Version2_21
	Version2_22 = nix.Version2_22
	Version2_23 = nix.Version2_23
	Version2_24 = nix.Version2_24
	Version2_25 = nix.Version2_25

	MinVersion = nix.Version2_12
)

type (
	Nix       = nix.Nix
	Cmd       = nix.Cmd
	Args      = nix.Args
	Info      = nix.Info
	Installer = nix.Installer
)

var Default = nix.Default

func AtLeast(version string) bool              { return nix.AtLeast(version) }
func Command(args ...any) *Cmd                 { return nix.Command(args...) }
func SourceProfile() (sourced bool, err error) { return nix.SourceProfile() }
func System() string                           { return nix.System() }
func Version() string                          { return nix.Version() }
