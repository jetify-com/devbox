package services

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"go.jetpack.io/devbox/internal/plugin"
)

func StartProcessManager(
	ctx context.Context,
	processComposePath string,
	services plugin.Services,
	processComposeFilePath string,
) error {
	flags := []string{"-p", "8280"}
	for _, s := range services {
		if file, hasComposeYaml := s.ProcessComposeYaml(); hasComposeYaml {
			flags = append(flags, "-f", file)
		}
	}
	if processComposeFilePath != "" {
		flags = append(flags, "-f", processComposeFilePath)
	}
	cmd := exec.Command(processComposePath, flags...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
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
