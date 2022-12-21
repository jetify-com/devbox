package python

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"go.jetpack.io/devbox/internal/pkgsuggest/suggestors"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

type SuggestorPoetry struct{}

// implements interface Suggestor (compile-time check)
var _ suggestors.Suggestor = (*SuggestorPoetry)(nil)

func (s *SuggestorPoetry) IsRelevant(srcDir string) bool {
	return plansdk.FileExists(filepath.Join(srcDir, "poetry.lock")) ||
		plansdk.FileExists(filepath.Join(srcDir, "pyproject.toml"))
}
func (s *SuggestorPoetry) Packages(srcDir string) []string {
	version := s.PythonVersion(srcDir)
	pythonPkg := fmt.Sprintf("python%s", version.MajorMinorConcatenated())

	return []string{
		pythonPkg,
		"poetry",
	}
}

// TODO: This can be generalized to all python planners
func (s *SuggestorPoetry) PythonVersion(srcDir string) *plansdk.Version {
	defaultVersion, _ := plansdk.NewVersion("3.10.6")
	project := s.PyProject(srcDir)

	if project == nil {
		return defaultVersion
	}

	if v, err := plansdk.NewVersion(project.Tool.Poetry.Dependencies.Python); err == nil {
		return v
	}
	return defaultVersion
}

type pyProject struct {
	Tool struct {
		Poetry struct {
			Name         string `toml:"name"`
			Dependencies struct {
				Python string `toml:"python"`
			} `toml:"dependencies"`
			Packages []struct {
				Include string `toml:"include"`
				From    string `toml:"from"`
			} `toml:"packages"`
			Scripts map[string]string `toml:"scripts"`
		} `toml:"poetry"`
	} `toml:"tool"`
}

func (s *SuggestorPoetry) PyProject(srcDir string) *pyProject {
	pyProjectPath := filepath.Join(srcDir, "pyproject.toml")
	content, err := os.ReadFile(pyProjectPath)
	if err != nil {
		return nil
	}
	proj := pyProject{}
	_ = toml.Unmarshal(content, &proj)
	return &proj
}
