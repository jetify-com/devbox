package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"go.jetpack.io/devbox/internal/plugin"
)

const (
	pidfile = ".devbox/process-compose.pid"
)

func StartProcessManager(
	ctx context.Context,
	processComposePath string,
	services plugin.Services,
	processComposeFilePath string,
	processComposeBackground bool,
) error {
	fmt.Printf("Running StartProcessManager with background %v\n", processComposeBackground)
	flags := []string{"-p", "8280"}
	for _, s := range services {
		if file, hasComposeYaml := s.ProcessComposeYaml(); hasComposeYaml {
			flags = append(flags, "-f", file)
		}
	}
	if processComposeFilePath != "" {
		flags = append(flags, "-f", processComposeFilePath)
	}
	if processComposeBackground {
		flags = append(flags, "-t=false")
	}
	// run the exec.Command in the background
	cmd := exec.Command(processComposePath, flags...)
	// Route stdout to /dev/null
	cmd.Stdout = nil
	cmd.Stderr = nil
	//run cmd in the background
	if processComposeBackground {
		if err := cmd.Start(); err != nil {
			return err
		}
		pid := cmd.Process.Pid
		if err := os.WriteFile(pidfile, []byte(fmt.Sprintf("%v", pid)), 0644); err != nil {
			return err
		}
		fmt.Print("Services started in the background. To stop them, run `devbox services stop`.\n")
		return nil
	}
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
