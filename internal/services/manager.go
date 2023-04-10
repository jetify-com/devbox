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

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/xdg"
)

const (
	processComposeLogfile = string(".devbox/compose.log")
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

type instance struct {
	Pid  int `json:"pid"`
	Port int `json:"port"`
}

type instanceMap = map[string]instance

type globalProcessComposeConfig struct {
	Instances instanceMap
	Path      string   `json:"-"`
	File      *os.File `json:"-"`
}

func globalProcessComposeJSONPath() (string, error) {
	path := xdg.DataSubpath(filepath.Join("devbox/global/"))
	return filepath.Join(path, "process-compose.json"), errors.WithStack(os.MkdirAll(path, 0755))
}

func readGlobalProcessComposeJSON(configPath string) globalProcessComposeConfig {
	config := globalProcessComposeConfig{Instances: map[string]instance{}}

	err := errors.WithStack(cuecfg.ParseFile(configPath, &config.Instances))
	if err != nil {
		return config
	}
	config.Path = configPath
	return config
}

func writeGlobalProcessComposeJSON(config globalProcessComposeConfig) error {
	// convert config to json using cue
	json, err := cuecfg.MarshalJSON(config.Instances)
	if err != nil {
		return fmt.Errorf("failed to convert config to json: %w", err)
	}

	// write json to file
	path, err := globalProcessComposeJSONPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	if err := os.WriteFile(path, json, 0666); err != nil {
		return fmt.Errorf("failed to write global config file: %w", err)
	}

	return nil
}

func getGlobalConfig() (globalProcessComposeConfig, error) {

	configPath, err := globalProcessComposeJSONPath()
	if err != nil {
		return globalProcessComposeConfig{}, fmt.Errorf("failed to get config path: %w", err)
	}

	globalConfigFile, err := os.OpenFile(configPath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return globalProcessComposeConfig{}, fmt.Errorf("failed to open config file: %w", err)
	}
	lockFile(globalConfigFile.Fd())
	defer unlockFile(globalConfigFile.Fd())

	config := readGlobalProcessComposeJSON(configPath)
	config.File = globalConfigFile
	return config, nil
}

func addInstance(projectDir string, projectConfig instance) error {
	config, err := getGlobalConfig()
	if err != nil {
		return err
	}

	lockFile(config.File.Fd())
	defer unlockFile(config.File.Fd())

	config.Instances[projectDir] = projectConfig
	err = writeGlobalProcessComposeJSON(config)
	if err != nil {
		return fmt.Errorf("failed to write global process-compose config")
	}

	return nil
}

func removeInstance(projectDir string) error {
	config, err := getGlobalConfig()
	if err != nil {
		return err
	}

	lockFile(config.File.Fd())
	defer config.File.Close()
	defer unlockFile(config.File.Fd())

	delete(config.Instances, projectDir)
	err = writeGlobalProcessComposeJSON(config)
	if err != nil {
		fmt.Println("failed to write global process-compose config")
	}

	return nil
}

func StartProcessManager(
	ctx context.Context,
	w io.Writer,
	requestedServices []string,
	availableServices Services,
	projectDir string,
	processComposeBinPath string,
	processComposeFilePath string,
	processComposeBackground bool,
) error {
	// Check if process-compose is already running

	if ProcessManagerIsRunning(projectDir) {
		return fmt.Errorf("process-compose is already running. To stop it, run `devbox services stop`")
	}

	config, err := getGlobalConfig()
	if err != nil {
		return err
	}

	// Get the port to use for this project
	port, available := getAvailablePort(config)
	if !available {
		return fmt.Errorf("no available ports to start process-compose. You should run `devbox services stop` in your projects to free up ports")
	}

	// Start building the process-compose command
	flags := []string{"-p", strconv.Itoa(port)}
	upCommand := []string{"up"}

	if len(requestedServices) > 0 {
		flags = append(requestedServices, flags...)
		flags = append(upCommand, flags...)
	}

	for _, s := range availableServices {
		if file, hasComposeYaml := s.ProcessComposeYaml(); hasComposeYaml {
			flags = append(flags, "-f", file)
		}
	}

	file := lookupProcessCompose(projectDir, processComposeFilePath)
	if file != "" {
		flags = append(flags, "-f", file)
	}

	if processComposeBackground {
		flags = append(flags, "-t=false")
		cmd := exec.Command(processComposeBinPath, flags...)
		return runProcessManagerInBackground(cmd, port, config, projectDir)
	}

	cmd := exec.Command(processComposeBinPath, flags...)
	return runProcessManagerInForeground(cmd, port, projectDir, w)
}

func runProcessManagerInForeground(cmd *exec.Cmd, port int, projectDir string, w io.Writer) error {

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process-compose: %w", err)
	}

	projectConfig := instance{
		Pid:  cmd.Process.Pid,
		Port: port,
	}

	err := addInstance(projectDir, projectConfig)
	if err != nil {
		return fmt.Errorf("failed to add instance to global config: %w", err)
	}

	err = cmd.Wait()
	if err != nil && err.Error() == "exit status 1" {
		fmt.Fprintln(w, "Process-compose was terminated remotely")
		return nil
	} else if err != nil {
		return err
	}
	return removeInstance(projectDir)
}

func runProcessManagerInBackground(cmd *exec.Cmd, port int, config globalProcessComposeConfig, projectDir string) error {

	logfile, err := os.OpenFile(processComposeLogfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("failed to open process-compose log file: %w", err)
	}

	cmd.Stdout = logfile
	cmd.Stderr = logfile

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process-compose: %w", err)
	}

	projectConfig := instance{
		Pid:  cmd.Process.Pid,
		Port: port,
	}

	config.Instances[projectDir] = projectConfig

	err = writeGlobalProcessComposeJSON(config)
	if err != nil {
		return fmt.Errorf("failed to write global process-compose config: %w", err)
	}

	return nil
}

func StopProcessManager(
	ctx context.Context,
	projectDir string,
	w io.Writer,
) error {

	config, err := getGlobalConfig()
	if err != nil {
		return err
	}
	defer config.File.Close()

	project, ok := config.Instances[projectDir]
	if !ok {
		return fmt.Errorf("process-compose is not running or it's config is missing. To start it, run `devbox services up`")
	}

	defer func() {
		err = removeInstance(projectDir)
	}()

	pid, _ := os.FindProcess(project.Pid)
	err = pid.Signal(os.Interrupt)
	if err != nil {
		return fmt.Errorf("process-compose is not running. To start it, run `devbox services up`")
	}

	fmt.Fprintf(w, "Process-compose stopped successfully.\n")
	return nil
}

func StopAllProcessManagers(ctx context.Context, w io.Writer) error {
	configPath, err := globalProcessComposeJSONPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Open the config file, defer closing to the end of the scope
	globalConfigFile, err := os.OpenFile(configPath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer globalConfigFile.Close()

	// Lock the config file, defer unlocking to the end of the scope

	lockFile(globalConfigFile.Fd())
	defer unlockFile(globalConfigFile.Fd())
	config := readGlobalProcessComposeJSON(configPath)

	for _, project := range config.Instances {
		pid, _ := os.FindProcess(project.Pid)
		err := pid.Signal(os.Interrupt)
		if err != nil {
			fmt.Printf("process-compose is not running. To start it, run `devbox services up`")
		}
	}

	config.Instances = make(map[string]instance)

	err = writeGlobalProcessComposeJSON(config)
	if err != nil {
		return fmt.Errorf("failed to write global process-compose config: %w", err)
	}

	return nil
}

func ProcessManagerIsRunning(projectDir string) bool {

	configPath, err := globalProcessComposeJSONPath()
	if err != nil {
		return false
	}

	config := readGlobalProcessComposeJSON(configPath)

	project, ok := config.Instances[projectDir]
	if !ok {
		return false
	}

	process, _ := os.FindProcess(project.Pid)

	err = process.Signal(syscall.Signal(0))
	if err != nil {
		fmt.Printf("Error: %s \n", err)
		delete(config.Instances, projectDir)
		_ = writeGlobalProcessComposeJSON(config)
		return false
	}

	return true
}

func GetProcessManagerPort(projectDir string) (int, error) {
	path, err := globalProcessComposeJSONPath()
	if err != nil {
		return 0, fmt.Errorf("process-compose is not running or it's config is missing. To start it, run `devbox services up`")
	}

	config := readGlobalProcessComposeJSON(path)

	project, ok := config.Instances[projectDir]
	if !ok {
		return 0, fmt.Errorf("process-compose is not running or it's config is missing. To start it, run `devbox services up`")
	}

	return project.Port, nil
}

func lockFile(fd uintptr) {

	lock := syscall.Flock_t{
		Type:   syscall.F_WRLCK,
		Whence: int16(io.SeekStart),
		Start:  0,
		Len:    0,
	}

	if err := syscall.FcntlFlock(fd, syscall.F_SETLK, &lock); err != nil {
		fmt.Printf("Error acquiring lock: %s \n", err)
	}
}

func unlockFile(fd uintptr) {
	lock := syscall.Flock_t{
		Type:   syscall.F_UNLCK,
		Whence: int16(io.SeekStart),
		Start:  0,
		Len:    0,
	}
	if err := syscall.FcntlFlock(fd, syscall.F_SETLK, &lock); err != nil {
		fmt.Printf("Error unlocking file: %s\n", err)
	}
}
