package pkgcfg

import (
	"embed"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

const pkgCfgDir = "package-configuration"

//go:embed package-configuration/*
var packageConfiguration embed.FS

func getConfig(pkg, rootDir string) (*config, error) {
	configFiles, err := packageConfiguration.ReadDir(pkgCfgDir)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Try to find perfect match first
	for _, file := range configFiles {
		if file.IsDir() {
			continue
		}
		if strings.Contains(strings.Split(file.Name(), ".")[0], pkg) {
			content, err := packageConfiguration.ReadFile(
				filepath.Join(pkgCfgDir, file.Name()),
			)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			cfg, err := buildConfig(&config{}, pkg, rootDir, string(content))
			if err != nil {
				return nil, errors.WithStack(err)
			}
			if cfg.Match != "" && !regexp.MustCompile(cfg.Match).MatchString(pkg) {
				continue
			}
			return cfg, nil
		}
	}
	return &config{}, nil
}

func getFileContent(cfg *config, contentPath string) ([]byte, error) {
	return packageConfiguration.ReadFile(filepath.Join(pkgCfgDir, contentPath))
}
