// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pullbox

import (
	"context"
	"net/url"
	"os"
	"path/filepath"

	"go.jetify.com/devbox/internal/cuecfg"
	"go.jetify.com/devbox/internal/devconfig"
	"go.jetify.com/devbox/internal/fileutil"
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

func (p *pullbox) pullTextDevboxConfig(ctx context.Context) error {
	if p.isLocalConfig() {
		return p.copyToProfile(p.URL)
	}

	cfg, err := devconfig.LoadConfigFromURL(ctx, p.URL)
	if err != nil {
		return err
	}

	tmpDir, err := fileutil.CreateDevboxTempDir()
	if err != nil {
		return err
	}
	if err = cfg.Root.SaveTo(tmpDir); err != nil {
		return err
	}

	return p.copyToProfile(tmpDir)
}

func (p *pullbox) isLocalConfig() bool {
	_, err := os.Stat(p.URL)
	return err == nil
}
