package detector

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
		return 1, nil
	}
	return 0, nil
}

func (d *PHPDetector) Packages(ctx context.Context) ([]string, error) {
	version, err := d.phpVersion()
	if err != nil {
		return nil, err
	}
	return []string{fmt.Sprintf("php@%s", version)}, nil
}

func (d *PHPDetector) phpVersion() (string, error) {
	require := d.composerJSON.Require

	if require["php"] == "" {
		return "latest", nil
	}
	// Remove the caret (^) if present
	version := strings.TrimPrefix(require["php"], "^")

	// Extract version in the format x, x.y, or x.y.z
	re := regexp.MustCompile(`^(\d+(\.\d+){0,2})`)
	match := re.FindString(version)
	if match == "" {
		return "latest", nil
	}

	version = match

	return version, nil
}

func (d *PHPDetector) phpExtensions() ([]string, error) {
	extensions := []string{}
	for key := range d.composerJSON.Require {
		if strings.HasPrefix(key, "ext-") {
			extensions = append(extensions, "phpExtensions."+strings.TrimPrefix(key, "ext-"))
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
