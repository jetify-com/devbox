package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"go.jetpack.io/devbox/internal/plugin"
)

func StartProcessManager(
	ctx context.Context,
	w io.Writer,
	processComposePath string,
	services plugin.Services,
	processComposeFilePath string,
	processComposePidfile string,
	processComposeBackground bool,
) error {
	//Open the pidfile
	if pid, err := os.ReadFile(processComposePidfile); err == nil {
		// If the pidfile exists, check if the process is running
		if _, err := os.FindProcess(int(pid[0])); err == nil {
			return fmt.Errorf("process-compose is already running. To stop it, run `devbox services stop`")
		}
	}

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
		if err := os.WriteFile(processComposePidfile, []byte(fmt.Sprintf("%v", pid)), 0644); err != nil {
			return err
		}
		fmt.Fprintf(w, "Services started in the background. To stop them, run `devbox services stop`.\n")
		return nil
	}
	return cmd.Run()
}

func StopProcessManager(
	ctx context.Context,
	w io.Writer,
	processComposePidfile string,
) error {
	var pidfile []byte
	var pid *os.Process

	pidfile, err := os.ReadFile(processComposePidfile)
	if err != nil {
		return fmt.Errorf("process-compose is not running. To start it, run `devbox services start`")
	}

	os.Remove(processComposePidfile)
	pidInt, err := strconv.Atoi(string(pidfile))
	if err != nil {
		return fmt.Errorf("invalid pid, removing pidfile")
	}

	pid, err = os.FindProcess(pidInt)
	if err != nil {
		return fmt.Errorf("process-compose is not running. To start it, run `devbox services start`")
	}

	err = pid.Signal(os.Interrupt)
	if err != nil {
		return fmt.Errorf("unable to stop process, please run `pkill process-compose` to terminate it manually with error: %v", err)
	}

	fmt.Fprintf(w, "Process-compose stopped successfully.\n")
	return nil
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
