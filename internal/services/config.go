package services

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/cuecfg"
)

type Process struct {
	Command  string `yaml:"command"`
	IsDaemon bool   `yaml:"is_daemon,omitempty"`
	Shutdown struct {
		Command        string `yaml:"command,omitempty"`
		TimeoutSeconds int    `yaml:"timeout_seconds,omitempty"`
		Signal         int    `yaml:"signal,omitempty"`
	} `yaml:"shutdown,omitempty"`
	DependsOn map[string]struct {
		Condition string `yaml:"condition,omitempty"`
	} `yaml:"depends_on,omitempty"`
	Availability struct {
		Restart string `yaml:"restart,omitempty"`
	} `yaml:"availability,omitempty"`
}

type ProcessComposeYaml struct {
	Version   string             `yaml:"version"`
	Processes map[string]Process `yaml:"processes"`
}

func ReadProcessCompose(path string) (Services, error) {
	processCompose := &ProcessComposeYaml{}
	services := Services{}
	errors := errors.WithStack(cuecfg.ParseFile(path, processCompose))
	if errors != nil {
		return nil, errors
	}

	for name := range processCompose.Processes {
		svc := Service{
			Name:               name,
			ProcessComposePath: path,
		}
		services[name] = svc
	}

	return services, nil
}

func LookupProcessCompose(projectDir, path string) string {
	if path == "" {
		path = projectDir
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(projectDir, path)
	}

	pathsToCheck := []string{
		path,
		filepath.Join(path, "process-compose.yaml"),
		filepath.Join(path, "process-compose.yml"),
	}

	for _, p := range pathsToCheck {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}

	return ""
}
