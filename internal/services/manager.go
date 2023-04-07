package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/xdg"
)

const (
	processComposeLogfile = string(".devbox/compose.log")
	processComposeUIDFile = string(".devbox/compose.uid")
)

func defaultPorts() [10]int {
	return [10]int{8260, 8261, 8262, 8263, 8264, 8265, 8266, 8267, 8268, 8269}
}

func getAvailablePort(config globalProcessComposeConfig) (int, bool) {
	ports := defaultPorts()
	for _, p := range ports {
		available := true
		for _, instance := range config.Instances {
			if instance.Port == p {
				available = false
			}
		}
		if available {
			return p, true
		}
	}
	return 0, false
}

type projectConfig struct {
	Pid  int `json:"pid"`
	Port int `json:"port"`
}

type globalProcessComposeConfig struct {
	Instances map[string]projectConfig `json:"instances"`
}

func globalProcessComposeConfigPath() (string, error) {
	path := xdg.DataSubpath(filepath.Join("devbox/global/"))
	return path, errors.WithStack(os.MkdirAll(path, 0755))
}

func readGlobalProcessComposeConfig() globalProcessComposeConfig {
	config := globalProcessComposeConfig{Instances: map[string]projectConfig{}}
	path, err := globalProcessComposeConfigPath()
	if err != nil {
		return config
	}
	path = filepath.Join(path, "process-compose.json")
	file, err := os.Open(path)
	if err != nil {
		return config
	}
	defer file.Close()

	err = errors.WithStack(cuecfg.ParseFile(path, &config))
	if err != nil {
		return config
	}
	return config
}

func writeGlobalProcessComposeConfig(config globalProcessComposeConfig) error {
	// convert config to json using cue
	json, err := cuecfg.MarshalJSON(config)
	if err != nil {
		return fmt.Errorf("failed to convert config to json: %w", err)
	}

	// write json to file
	path, err := globalProcessComposeConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	path = filepath.Join(path, "process-compose.json")

	if err = os.WriteFile(path, json, 0666); err != nil {
		return fmt.Errorf("failed to open process-compose log file: %w", err)
	}

	return nil
}

func cleanupProject(globalProcessComposeConfig globalProcessComposeConfig, runID string) {
	os.Remove(processComposeUIDFile)
	delete(globalProcessComposeConfig.Instances, runID)
	err := writeGlobalProcessComposeConfig(globalProcessComposeConfig)
	if err != nil {
		fmt.Println("failed to write global process-compose config")
	}
}

func StartProcessManager(
	ctx context.Context,
	w io.Writer,
	requestedServices []string,
	services Services,
	projectDir string,
	processComposePath string,
	processComposeFilePath string,
	processComposeBackground bool,
) error {
	// Check if process-compose is already running

	if ProcessManagerIsRunning() {
		return fmt.Errorf("process-compose is already running. To stop it, run `devbox services stop`")
	}

	port, available := getAvailablePort(readGlobalProcessComposeConfig())
	if !available {
		return fmt.Errorf("no available ports to start process-compose. You should run `devbox services stop` in your projects to free up ports")
	}

	//convert port to string

	flags := []string{"-p", strconv.Itoa(port)}
	upCommand := []string{"up"}

	if len(requestedServices) > 0 {
		// append requested services and flags to 'up'
		flags = append(requestedServices, flags...)
		flags = append(upCommand, flags...)
	}

	for _, s := range services {
		if file, hasComposeYaml := s.ProcessComposeYaml(); hasComposeYaml {
			flags = append(flags, "-f", file)
		}
	}

	file := lookupProcessCompose(projectDir, processComposeFilePath)
	if file != "" {
		flags = append(flags, "-f", file)
	}

	//run cmd in the background
	if processComposeBackground {
		flags = append(flags, "-t=false")
		cmd := exec.Command(processComposePath, flags...)
		return runProcessManagerInBackground(cmd, port)
	}

	cmd := exec.Command(processComposePath, flags...)
	return runProcessManagerInForeground(cmd, port, w)
}

func runProcessManagerInForeground(cmd *exec.Cmd, port int, w io.Writer) error {
	runID := uuid.New().String()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process-compose: %w", err)
	}

	globalProcessComposeConfig := readGlobalProcessComposeConfig()

	projectConfig := projectConfig{
		Pid:  cmd.Process.Pid,
		Port: port,
	}

	globalProcessComposeConfig.Instances[runID] = projectConfig

	if err := os.WriteFile(processComposeUIDFile, []byte(runID), 0666); err != nil {
		return fmt.Errorf("failed to write uidfile: %w", err)
	}

	err := writeGlobalProcessComposeConfig(globalProcessComposeConfig)
	if err != nil {
		return fmt.Errorf("failed to write global process-compose config: %w", err)
	}

	defer cleanupProject(globalProcessComposeConfig, runID)
	err = cmd.Wait()
	if err != nil && err.Error() == "exit status 1" {
		fmt.Fprintln(w, "Process-compose was terminated remotely")
		return nil
	}
	return err
}

func runProcessManagerInBackground(cmd *exec.Cmd, port int) error {
	runID := uuid.New().String()

	logfile, err := os.OpenFile(processComposeLogfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("failed to open process-compose log file: %w", err)
	}

	cmd.Stdout = logfile
	cmd.Stderr = logfile

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process-compose: %w", err)
	}

	globalProcessComposeConfig := readGlobalProcessComposeConfig()

	projectConfig := projectConfig{
		Pid:  cmd.Process.Pid,
		Port: port,
	}

	globalProcessComposeConfig.Instances[runID] = projectConfig

	err = writeGlobalProcessComposeConfig(globalProcessComposeConfig)
	if err != nil {
		return fmt.Errorf("failed to write global process-compose config: %w", err)
	}

	if err := os.WriteFile(processComposeUIDFile, []byte(runID), 0666); err != nil {
		return fmt.Errorf("failed to write uidfile: %w", err)
	}

	return nil
}

func StopProcessManager(
	ctx context.Context,
	w io.Writer,
) error {
	globalProcessComposeConfig := readGlobalProcessComposeConfig()

	uid, err := os.ReadFile(processComposeUIDFile)
	if err != nil {
		return fmt.Errorf("process-compose is not running or it's config is missing. To start it, run `devbox services up`")
	}

	project, ok := globalProcessComposeConfig.Instances[string(uid)]
	if !ok {
		return fmt.Errorf("process-compose is not running or it's config is missing. To start it, run `devbox services up`")
	}

	defer cleanupProject(globalProcessComposeConfig, string(uid))

	pid, _ := os.FindProcess(project.Pid)
	err = pid.Signal(os.Interrupt)
	if err != nil {
		return fmt.Errorf("process-compose is not running. To start it, run `devbox services up`")
	}

	fmt.Fprintf(w, "Process-compose stopped successfully.\n")
	return nil
}

func StopAllProcessManagers(ctx context.Context, w io.Writer) error {
	globalProcessComposeConfig := readGlobalProcessComposeConfig()

	for _, project := range globalProcessComposeConfig.Instances {
		pid, _ := os.FindProcess(project.Pid)
		err := pid.Signal(os.Interrupt)
		if err != nil {
			fmt.Printf("process-compose is not running. To start it, run `devbox services up`")
		}
	}

	globalProcessComposeConfig.Instances = make(map[string]projectConfig)

	err := writeGlobalProcessComposeConfig(globalProcessComposeConfig)
	if err != nil {
		return fmt.Errorf("failed to write global process-compose config: %w", err)
	}

	return nil
}

func ProcessManagerIsRunning() bool {
	config := readGlobalProcessComposeConfig()

	data, err := os.ReadFile(processComposeUIDFile)
	if err != nil {
		return false
	}

	uid := string(data)
	project, ok := config.Instances[uid]
	if !ok {
		os.Remove(processComposeUIDFile)
		return false
	}

	process, _ := os.FindProcess(project.Pid)

	err = process.Signal(syscall.Signal(0))
	if err != nil {
		fmt.Printf("Error: %s \n", err)
		os.Remove(processComposeUIDFile)
		delete(config.Instances, uid)
		_ = writeGlobalProcessComposeConfig(config)
		return false
	}

	return true
}

func GetProcessManagerPort() (int, error) {
	config := readGlobalProcessComposeConfig()

	uid, err := os.ReadFile(processComposeUIDFile)
	if err != nil {
		return 0, err
	}

	project, ok := config.Instances[string(uid)]
	if !ok {
		os.Remove(processComposeUIDFile)
		return 0, fmt.Errorf("process-compose is not running or it's config is missing. To start it, run `devbox services up`")
	}

	return project.Port, nil
}
