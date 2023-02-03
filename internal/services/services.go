package services

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/plugin"
)

func Start(pkgs, serviceNames []string, projectDir string, w io.Writer) error {
	return toggleServices(pkgs, serviceNames, projectDir, w, startService)
}

func Stop(pkgs, serviceNames []string, projectDir string, w io.Writer) error {
	return toggleServices(pkgs, serviceNames, projectDir, w, stopService)
}

type serviceAction int

const (
	startService serviceAction = iota
	stopService
)

func toggleServices(
	pkgs,
	serviceNames []string,
	projectDir string,
	w io.Writer,
	action serviceAction,
) error {
	services, err := plugin.GetServices(pkgs, projectDir)
	if err != nil {
		return err
	}
	envVars, err := plugin.Env(pkgs, projectDir)
	if err != nil {
		return err
	}
	for _, name := range serviceNames {
		service, found := services[name]
		if !found {
			return usererr.New("Service not found")
		}
		cmd := exec.Command(
			"sh",
			"-c",
			lo.Ternary(action == startService, service.Start, service.Stop),
		)
		cmd.Stdout = w
		cmd.Stderr = w
		cmd.Env = envVars
		cmd.Env = append(cmd.Env, os.Environ()...)
		if err = cmd.Run(); err != nil {
			actionString := lo.Ternary(action == startService, "start", "stop")
			if len(serviceNames) == 1 {
				return usererr.WithUserMessage(err, "Service %q failed to %s", name, actionString)
			}
			fmt.Fprintf(w, "Service %q failed to %s. Error = %s\n", name, actionString, err)
		} else {
			actionStringPast := lo.Ternary(action == startService, "started", "stopped")
			fmt.Fprintf(w, "Service %q %s\n", name, actionStringPast)
			port, err := service.Port()
			if err != nil {
				fmt.Fprintf(w, "Error getting port: %s\n", err)
			}
			if err := updateServiceStatus(projectDir, &serviceStatus{
				Name:    name,
				Port:    port,
				Running: action == startService,
			}); err != nil {
				fmt.Fprintf(w, "Error updating status file: %s\n", err)
			}
		}
	}
	return nil
}
