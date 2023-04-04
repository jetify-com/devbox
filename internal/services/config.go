package services

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/cuecfg"
)

func FromProcessComposeYaml(projectDir string) Services {
	// TODO need to handle if a filepath is passed in
	if processComposeYaml := lookupProcessCompose(projectDir, ""); processComposeYaml != "" {
		userSvcs, err := readProcessCompose(processComposeYaml)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading process-compose.yaml: %s, skipping", err)
			return nil
		}
		return userSvcs
	}
	return Services{}
}

func readProcessCompose(path string) (Services, error) {
	processCompose := &types.Project{}
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

func lookupProcessCompose(projectDir, path string) string {
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
