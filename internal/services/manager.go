package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

func StartProcessManager(
	ctx context.Context,
	w io.Writer,
	requestedServices []string,
	processComposePath string,
	services Services,
	processComposeFilePath string,
	processComposePidfile string,
	processComposeLogfile string,
	processComposeBackground bool,
) error {
	// Check if process-compose is already running

	if ProcessManagerIsRunning(processComposePidfile) {
		return fmt.Errorf("process-compose is already running. To stop it, run `devbox services stop`")
	}

	flags := []string{"-p", "8280"}
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

	if processComposeFilePath != "" {
		flags = append(flags, "-f", processComposeFilePath)
	}

	//run cmd in the background
	if processComposeBackground {
		flags = append(flags, "-t=false")
		cmd := exec.Command(processComposePath, flags...)
		return runProcessManagerInBackground(cmd, processComposePidfile, processComposeLogfile)
	}

	cmd := exec.Command(processComposePath, flags...)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process-compose: %w", err)
	}

	if err := os.WriteFile(processComposePidfile, []byte(strconv.Itoa(cmd.Process.Pid)), 0666); err != nil {
		return fmt.Errorf("failed to write pidfile: %w", err)
	}

	err := cmd.Wait()
	// Cleanup
	fmt.Print(processComposePidfile)
	os.Remove(processComposePidfile)
	return err
}

func runProcessManagerInBackground(
	cmd *exec.Cmd,
	processComposePidfile,
	processComposeLogfile string,
) error {

	logfile, err := os.OpenFile(processComposeLogfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open process-compose log file: %w", err)
	}

	cmd.Stdout = logfile
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process-compose: %w", err)
	}

	if err := os.WriteFile(processComposePidfile, []byte(strconv.Itoa(cmd.Process.Pid)), 0666); err != nil {
		return fmt.Errorf("failed to write pidfile: %w", err)
	}

	return nil
}

func StopProcessManager(
	ctx context.Context,
	w io.Writer,
	processComposePidfile string,
) error {
	var pidfile []byte
	var pid *os.Process

	pidfile, err := os.ReadFile(processComposePidfile)
	fmt.Println(string(pidfile))
	if err != nil {
		return fmt.Errorf("process-compose is not running. To start it, run `devbox services start`")
	}

	os.Remove(processComposePidfile)
	pidInt, err := strconv.Atoi(string(pidfile))
	if err != nil {
		return fmt.Errorf("invalid pid, removing pidfile")
	}

	pid, _ = os.FindProcess(pidInt)
	err = pid.Signal(os.Interrupt)
	if err != nil {
		fmt.Println(string(pidfile))
		return fmt.Errorf("process-compose is not running. To start it, run `devbox services start`")
	}

	fmt.Fprintf(w, "Process-compose stopped successfully.\n")
	return nil
}

func ProcessManagerIsRunning(processComposePidfile string) bool {
	pid, err := os.ReadFile(processComposePidfile)
	if err != nil {
		return false
	}

	pidToInt, _ := strconv.Atoi(string(pid))
	process, err := os.FindProcess(pidToInt)
	if err != nil {
		fmt.Printf("Error: %s \n", err)
		return false
	}

	err = process.Signal(syscall.Signal(0))
	if err != nil {
		fmt.Printf("Error: %s \n", err)
		return false
	}
	return true
}
