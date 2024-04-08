// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

// Package nix provides Go API for nix.
// Internally this is a wrapper around the nix command line utilities.
// I'd love to use a go SDK instead, and drop the dependency on the CLI.
// The dependency means that users need to install nix, before using devbox.
// Unfortunately, that go sdk does not exist. We would have to implement it.
package nix
