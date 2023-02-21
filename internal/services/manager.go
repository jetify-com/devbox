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
	globalBinPath string,
	services plugin.Services,
) error {
	flags := []string{"-p", "8280"}
	for _, s := range services {
		if file, hasComposeYaml := s.ProcessComposeYaml(); hasComposeYaml {
			flags = append(flags, "-f", file)
		}
	}
	cmd := exec.Command(filepath.Join(globalBinPath, "process-compose"), flags...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
