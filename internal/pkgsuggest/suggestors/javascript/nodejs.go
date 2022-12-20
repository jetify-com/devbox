package javascript

import (
	"path/filepath"

	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/pkgsuggest/suggestors"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

type Suggestor struct{}

// implements interface Suggestor (compile-time check)
var _ suggestors.Suggestor = (*Suggestor)(nil)

func (s *Suggestor) IsRelevant(srcDir string) bool {
	packageJSONPath := filepath.Join(srcDir, "package.json")
	return plansdk.FileExists(packageJSONPath)
}

func (s *Suggestor) Packages(srcDir string) []string {
	pkgManager := s.packageManager(srcDir)
	project := s.nodeProject(srcDir)
	packages := s.packages(pkgManager, project)

	return packages
}

type nodeProject struct {
	Scripts struct {
		Build string `json:"build,omitempty"`
		Start string `json:"start,omitempty"`
	}
	Engines struct {
		Node string `json:"node,omitempty"`
	} `json:"engines,omitempty"`
}

var versionMap = map[string]string{
	// Map node versions to the corresponding nixpkgs:
	"10": "nodejs-10_x",
	"12": "nodejs-12_x",
	"16": "nodejs-16_x",
	"18": "nodejs-18_x",
}
var defaultNodeJSPkg = "nodejs"

func (s *Suggestor) nodePackage(project *nodeProject) string {
	v := s.nodeVersion(project)
	if v != nil {
		pkg, ok := versionMap[v.Major()]
		if ok {
			return pkg
		}
	}

	return defaultNodeJSPkg
}

func (s *Suggestor) nodeVersion(project *nodeProject) *plansdk.Version {
	if s != nil {
		if v, err := plansdk.NewVersion(project.Engines.Node); err == nil {
			return v
		}
	}

	return nil
}

func (s *Suggestor) packageManager(srcDir string) string {
	yarnPkgLockPath := filepath.Join(srcDir, "yarn.lock")
	if plansdk.FileExists(yarnPkgLockPath) {
		return "yarn"
	}
	return "npm"
}

func (s *Suggestor) packages(pkgManager string, project *nodeProject) []string {
	nodeJSPkg := s.nodePackage(project)
	pkgs := []string{nodeJSPkg}

	if pkgManager == "yarn" {
		return append(pkgs, "yarn")
	}
	return pkgs
}

func (s *Suggestor) nodeProject(srcDir string) *nodeProject {
	packageJSONPath := filepath.Join(srcDir, "package.json")
	project := &nodeProject{}
	_ = cuecfg.ParseFile(packageJSONPath, project)

	return project
}
