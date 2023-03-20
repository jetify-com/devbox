package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cloud/envir"
	"go.jetpack.io/devbox/internal/plugin"
)

func Start(ctx context.Context, pkgs, serviceNames []string, projectDir string, w io.Writer) error {
	return toggleServices(ctx, pkgs, serviceNames, projectDir, w, startService)
}

func Stop(ctx context.Context, pkgs, serviceNames []string, projectDir string, w io.Writer) error {
	return toggleServices(ctx, pkgs, serviceNames, projectDir, w, stopService)
}

type serviceAction int

const (
	startService serviceAction = iota
	stopService
)

func toggleServices(
	ctx context.Context,
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
	contextChannels := []<-chan struct{}{}
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
			if port != "" {
				// Wait 5 seconds for each port forwarding to start. The function may
				// cancel the context earlier if it detects it already started
				childCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				contextChannels = append(contextChannels, childCtx.Done())
				if err := listenToAutoPortForwardingChangesOnRemote(childCtx, name, w, projectDir, action, cancel); err != nil {
					fmt.Fprintf(w, "Error listening to port forwarding changes: %s\n", err)
				}
			}
			if err := updateServiceStatusOnRemote(projectDir, &ServiceStatus{
				Name:    name,
				Port:    port,
				Running: action == startService,
			}); err != nil {
				fmt.Fprintf(w, "Error updating status file: %s\n", err)
			}
		}
	}

	for _, c := range contextChannels {
		<-c
	}

	if action == startService {
		return printProxyURL(w, lo.PickByKeys(services, serviceNames))
	}

	return nil
}

func listenToAutoPortForwardingChangesOnRemote(
	ctx context.Context,
	serviceName string,
	w io.Writer,
	projectDir string,
	action serviceAction,
	cancel context.CancelFunc,
) error {

	if !envir.IsCLICloudShell() {
		return nil
	}

	hostname, err := os.Hostname()
	if err != nil {
		return errors.WithStack(err)
	}

	fmt.Fprintf(w, "Waiting for port forwarding to start/stop for service %q\n", serviceName)

	// Listen to changes in the service status file
	return ListenToChanges(
		ctx,
		&ListenerOpts{
			HostID:     hostname,
			ProjectDir: projectDir,
			Writer:     w,
			UpdateFunc: func(service *ServiceStatus) (*ServiceStatus, bool) {
				if service == nil || service.Name != serviceName {
					return service, false
				}
				if action == startService && service.Running && service.Port != "" && service.LocalPort != "" {
					color.New(color.FgYellow).Fprintf(w, "Port forwarding %s:%s -> %s:%s\n", hostname, service.Port, "http://localhost", service.LocalPort)
					cancel()
				}
				if action == stopService && !service.Running && service.Port != "" {
					color.New(color.FgYellow).Fprintf(w, "Port forwarding %s:%s -> localhost stopped\n", hostname, service.Port)
					cancel()
				}
				return service, false
			},
		},
	)
}

func printProxyURL(w io.Writer, services plugin.Services) error {

	if !envir.IsDevboxCloud() {
		return nil
	}

	hostname, err := os.Hostname()
	if err != nil {
		return errors.WithStack(err)
	}

	printGeneric := false
	for _, service := range services {
		if port, _ := service.Port(); port != "" {
			color.New(color.FgHiGreen).Fprintf(
				w,
				"To access %s on this vm use: %s-%s.svc.devbox.sh\n",
				service.Name,
				hostname,
				port,
			)
		} else {
			printGeneric = true
		}
	}

	if printGeneric {
		color.New(color.FgHiGreen).Fprintf(
			w,
			"To access other services on this vm use: %s-<port>.svc.devbox.sh\n",
			hostname,
		)
	}
	return nil
}
