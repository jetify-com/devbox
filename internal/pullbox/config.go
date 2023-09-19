// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pullbox

import (
	"net/url"
	"os"
	"path/filepath"

	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/devconfig"
	"go.jetpack.io/devbox/internal/fileutil"
)

func (p *pullbox) IsTextDevboxConfig() bool {
	if u, err := url.Parse(p.URL); err == nil {
		ext := filepath.Ext(u.Path)
		return cuecfg.IsSupportedExtension(ext)
	}
	// For invalid URLS, just look at the extension
	ext := filepath.Ext(p.URL)
	return cuecfg.IsSupportedExtension(ext)
}

func (p *pullbox) pullTextDevboxConfig() error {
	if p.isLocalConfig() {
		return p.copyToProfile(p.URL)
	}

	cfg, err := devconfig.LoadConfigFromURL(p.URL)
	if err != nil {
		return err
	}

	tmpDir, err := fileutil.CreateDevboxTempDir()
	if err != nil {
		return err
	}
	if err = cfg.SaveTo(tmpDir); err != nil {
		return err
	}

	return p.copyToProfile(tmpDir)
}

func (p *pullbox) isLocalConfig() bool {
	_, err := os.Stat(p.URL)
	return err == nil
}
