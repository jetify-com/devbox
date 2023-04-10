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

type projectConfig struct {
	Pid  int `json:"pid"`
	Port int `json:"port"`
}

type projectMap = map[string]projectConfig

type globalProcessComposeConfig struct {
	Instances  projectMap
	GlobalPath string   `json:"-"`
	FileRef    *os.File `json:"-"`
}

func (c *globalProcessComposeConfig) updateFromFile(path string) {
	// read the config from file
	config := readGlobalProcessComposeConfig(path)

	// update the config
	c.Instances = config.Instances
	c.GlobalPath = config.GlobalPath
}

func globalProcessComposeConfigPath() (string, error) {
	path := xdg.DataSubpath(filepath.Join("devbox/global/"))
	return filepath.Join(path, "process-compose.json"), errors.WithStack(os.MkdirAll(path, 0755))
}

func readGlobalProcessComposeConfig(configPath string) globalProcessComposeConfig {
	config := globalProcessComposeConfig{Instances: map[string]projectConfig{}}

	err := errors.WithStack(cuecfg.ParseFile(configPath, &config.Instances))
	if err != nil {
		return config
	}
	config.GlobalPath = configPath
	return config
}

func writeGlobalProcessComposeConfig(config globalProcessComposeConfig) error {
	// convert config to json using cue
	json, err := cuecfg.MarshalJSON(config.Instances)
	if err != nil {
		return fmt.Errorf("failed to convert config to json: %w", err)
	}

	// write json to file
	path, err := globalProcessComposeConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	if err := os.WriteFile(path, json, 0666); err != nil {
		return fmt.Errorf("failed to write global config file: %w", err)
	}

	return nil
}

func cleanupProject(config globalProcessComposeConfig, projectDir string) {
	fmt.Printf("Cleaning up project with FD: %d", config.FileRef.Fd())
	lockFile(config.FileRef.Fd())
	defer unlockFile(config.FileRef.Fd())

	delete(config.Instances, projectDir)
	err := writeGlobalProcessComposeConfig(config)
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

	if ProcessManagerIsRunning(projectDir) {
		return fmt.Errorf("process-compose is already running. To stop it, run `devbox services stop`")
	}

	// Get the right path, based on the user's XDG settings
	configPath, err := globalProcessComposeConfigPath()
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

	//Read the file, store the ref in the config
	config := readGlobalProcessComposeConfig(configPath)
	config.FileRef = globalConfigFile

	// Get the port to use for this project
	port, available := getAvailablePort(config)
	if !available {
		return fmt.Errorf("no available ports to start process-compose. You should run `devbox services stop` in your projects to free up ports")
	}

	// Now we have everything we need, let's start building the process-compose command
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
		return runProcessManagerInBackground(cmd, port, config, projectDir)
	}

	cmd := exec.Command(processComposePath, flags...)
	return runProcessManagerInForeground(cmd, port, config, projectDir, w)
}

func runProcessManagerInForeground(cmd *exec.Cmd, port int, config globalProcessComposeConfig, projectDir string, w io.Writer) error {

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process-compose: %w", err)
	}

	projectConfig := projectConfig{
		Pid:  cmd.Process.Pid,
		Port: port,
	}

	config.Instances[projectDir] = projectConfig

	err := writeGlobalProcessComposeConfig(config)
	if err != nil {
		return fmt.Errorf("failed to write global process-compose config: %w", err)
	}

	unlockFile(config.FileRef.Fd())
	err = cmd.Wait()
	config.updateFromFile(config.GlobalPath)
	if err != nil && err.Error() == "exit status 1" {
		fmt.Fprintln(w, "Process-compose was terminated remotely")
		return nil
	}
	cleanupProject(config, projectDir)
	return err
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

	projectConfig := projectConfig{
		Pid:  cmd.Process.Pid,
		Port: port,
	}

	config.Instances[projectDir] = projectConfig

	err = writeGlobalProcessComposeConfig(config)
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
	configPath, err := globalProcessComposeConfigPath()
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

	config := readGlobalProcessComposeConfig(configPath)
	config.FileRef = globalConfigFile

	project, ok := config.Instances[string(projectDir)]
	if !ok {
		return fmt.Errorf("process-compose is not running or it's config is missing. To start it, run `devbox services up`")
	}

	defer cleanupProject(config, projectDir)

	pid, _ := os.FindProcess(project.Pid)
	err = pid.Signal(os.Interrupt)
	if err != nil {
		return fmt.Errorf("process-compose is not running. To start it, run `devbox services up`")
	}

	fmt.Fprintf(w, "Process-compose stopped successfully.\n")
	return nil
}

func StopAllProcessManagers(ctx context.Context, w io.Writer) error {
	configPath, err := globalProcessComposeConfigPath()
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

	config := readGlobalProcessComposeConfig(configPath)

	for _, project := range config.Instances {
		pid, _ := os.FindProcess(project.Pid)
		err := pid.Signal(os.Interrupt)
		if err != nil {
			fmt.Printf("process-compose is not running. To start it, run `devbox services up`")
		}
	}

	config.Instances = make(map[string]projectConfig)

	err = writeGlobalProcessComposeConfig(config)
	if err != nil {
		return fmt.Errorf("failed to write global process-compose config: %w", err)
	}

	return nil
}

func ProcessManagerIsRunning(projectDir string) bool {

	configPath, err := globalProcessComposeConfigPath()
	if err != nil {
		return false
	}

	config := readGlobalProcessComposeConfig(configPath)

	project, ok := config.Instances[projectDir]
	if !ok {
		return false
	}

	process, _ := os.FindProcess(project.Pid)

	err = process.Signal(syscall.Signal(0))
	if err != nil {
		fmt.Printf("Error: %s \n", err)
		delete(config.Instances, projectDir)
		_ = writeGlobalProcessComposeConfig(config)
		return false
	}

	return true
}

func GetProcessManagerPort(projectDir string) (int, error) {
	path, err := globalProcessComposeConfigPath()
	if err != nil {
		return 0, fmt.Errorf("process-compose is not running or it's config is missing. To start it, run `devbox services up`")
	}

	config := readGlobalProcessComposeConfig(path)

	project, ok := config.Instances[string(projectDir)]
	if !ok {
		return 0, fmt.Errorf("process-compose is not running or it's config is missing. To start it, run `devbox services up`")
	}

	return project.Port, nil
}

func lockFile(fd uintptr) error {

	lock := syscall.Flock_t{
		Type:   syscall.F_WRLCK,
		Whence: int16(os.SEEK_SET),
		Start:  0,
		Len:    0,
	}

	if err := syscall.FcntlFlock(fd, syscall.F_SETLK, &lock); err != nil {
		fmt.Printf("Error acquiring lock: %s", err)
		return nil
	}
	return nil
}

func unlockFile(fd uintptr) error {
	lock := syscall.Flock_t{
		Type:   syscall.F_UNLCK,
		Whence: int16(os.SEEK_SET),
		Start:  0,
		Len:    0,
	}
	if err := syscall.FcntlFlock(fd, syscall.F_SETLK, &lock); err != nil {
		fmt.Printf("Error unlocking file: %s", err)
		return nil
	}
	return nil

}
