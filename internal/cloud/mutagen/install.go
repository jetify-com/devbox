// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package mutagen

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/cavaliergopher/grab/v3"

	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/fileutil"
)

func InstallMutagenOnce(binPath string) error {
	if fileutil.IsFile(binPath) {
		// Already installed, do nothing
		// TODO: ideally we would check that the right version
		//   is installed, and maybe we should also validate
		//   with a checksum.
		return nil
	}

	url := mutagenURL()
	installDir := filepath.Dir(binPath)

	return Install(url, installDir)
}

func Install(url, installDir string) error {
	debug.Log("installing mutagen from %s to %s", url, installDir)
	err := os.MkdirAll(installDir, 0o755)
	if err != nil {
		return err
	}

	// TODO: add checksum validation
	resp, err := grab.Get(os.TempDir(), url)
	if err != nil {
		return err
	}

	tarPath := resp.Filename
	tarReader, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	return fileutil.Untar(tarReader, installDir)
}

func mutagenURL() string {
	repo := "mutagen-io/mutagen"
	pkg := "mutagen"
	version := "v0.16.2" // Hard-coded for now, but change to always get the latest?
	platform := detectPlatform()

	return fmt.Sprintf("https://github.com/%s/releases/download/%s/%s_%s_%s.tar.gz", repo, version, pkg, platform, version)
}

func detectOS() string {
	return runtime.GOOS
}

func detectArch() string {
	return runtime.GOARCH
}

func detectPlatform() string {
	os := detectOS()
	arch := detectArch()
	return fmt.Sprintf("%s_%s", os, arch)
}
