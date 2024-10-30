package detector

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
)

type packageJSON struct {
	Engines struct {
		Node string `json:"node"`
	} `json:"engines"`
}

type NodeJSDetector struct {
	Root        string
	packageJSON *packageJSON
}

var _ Detector = &NodeJSDetector{}

func (d *NodeJSDetector) Init() error {
	pkgJSON, err := loadPackageJSON(d.Root)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	d.packageJSON = pkgJSON
	return nil
}

func (d *NodeJSDetector) Relevance(path string) (float64, error) {
	if d.packageJSON == nil {
		return 0, nil
	}
	return 1, nil
}

func (d *NodeJSDetector) Packages(ctx context.Context) ([]string, error) {
	return []string{"nodejs@" + d.nodeVersion(ctx)}, nil
}

func (d *NodeJSDetector) nodeVersion(ctx context.Context) string {
	if d.packageJSON == nil || d.packageJSON.Engines.Node == "" {
		return "latest" // Default to latest if not specified
	}

	// Remove any non-semver characters (e.g. ">=", "^", etc)
	version := "latest"
	semverRegex := regexp.MustCompile(`\d+(\.\d+)?(\.\d+)?`)
	if match := semverRegex.FindString(d.packageJSON.Engines.Node); match != "" {
		version = match
	}

	return determineBestVersion(ctx, "nodejs", version)
}

func loadPackageJSON(root string) (*packageJSON, error) {
	path := filepath.Join(root, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}
