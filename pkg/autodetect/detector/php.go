package detector

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.jetify.com/devbox/internal/searcher"
)

type composerJSON struct {
	Require map[string]string `json:"require"`
}

type PHPDetector struct {
	Root         string
	composerJSON *composerJSON
}

var _ Detector = &PHPDetector{}

func (d *PHPDetector) Init() error {
	composer, err := loadComposerJSON(d.Root)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	d.composerJSON = composer
	return nil
}

func (d *PHPDetector) Relevance(path string) (float64, error) {
	if d.composerJSON == nil {
		return 0, nil
	}
	return 1, nil
}

func (d *PHPDetector) Packages(ctx context.Context) ([]string, error) {
	packages := []string{fmt.Sprintf("php@%s", d.phpVersion(ctx))}
	extensions, err := d.phpExtensions(ctx)
	if err != nil {
		return nil, err
	}
	packages = append(packages, extensions...)
	return packages, nil
}

func (d *PHPDetector) Env(ctx context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}

func (d *PHPDetector) phpVersion(ctx context.Context) string {
	require := d.composerJSON.Require

	if require["php"] == "" {
		return "latest"
	}
	// Remove the caret (^) if present
	version := strings.TrimPrefix(require["php"], "^")

	// Extract version in the format x, x.y, or x.y.z
	re := regexp.MustCompile(`^(\d+(\.\d+){0,2})`)
	match := re.FindString(version)

	return determineBestVersion(ctx, "php", match)
}

func (d *PHPDetector) phpExtensions(ctx context.Context) ([]string, error) {
	resolved, err := searcher.Client().ResolveV2(ctx, "php", d.phpVersion(ctx))
	if err != nil {
		return nil, err
	}

	// extract major-minor from resolved.Version
	re := regexp.MustCompile(`^(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(resolved.Version)
	if len(matches) < 3 {
		return nil, fmt.Errorf("could not parse PHP version: %s", resolved.Version)
	}
	majorMinor := matches[1] + matches[2]

	extensions := []string{}
	for key := range d.composerJSON.Require {
		if strings.HasPrefix(key, "ext-") {
			// The way nix versions php extensions is inconsistent. Sometimes the version is the PHP
			// version, sometimes it's the extension version. We just use @latest everywhere which in
			// practice will just use the version of the extension that exists in the same nixpkgs as
			// the php version.
			extensions = append(
				extensions,
				fmt.Sprintf("php%sExtensions.%s@latest", majorMinor, strings.TrimPrefix(key, "ext-")),
			)
		}
	}

	return extensions, nil
}

func loadComposerJSON(root string) (*composerJSON, error) {
	composerPath := filepath.Join(root, "composer.json")
	composerData, err := os.ReadFile(composerPath)
	if err != nil {
		return nil, err
	}
	var composer composerJSON
	err = json.Unmarshal(composerData, &composer)
	if err != nil {
		return nil, err
	}
	return &composer, nil
}
