package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
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
	envVars, err := plugin.Env(pkgs, projectDir)
	if err != nil {
		return err
	}
	var waitGroup sync.WaitGroup
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
			if port != "" {
				if err := listenToAutoPortForwardingChangesOnRemote(ctx, name, w, projectDir, action, &waitGroup); err != nil {
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

	waitGroup.Wait()
	return nil
}

func listenToAutoPortForwardingChangesOnRemote(
	ctx context.Context,
	serviceName string,
	w io.Writer,
	projectDir string,
	action serviceAction,
	waitGroup *sync.WaitGroup,
) error {

	if os.Getenv("DEVBOX_REGION") == "" {
		return nil
	}
	hostname, err := hostname()
	if err != nil {
		return errors.WithStack(err)
	}

	fmt.Fprintf(w, "Waiting for port forwarding to start/stop for service %q\n", serviceName)
	waitGroup.Add(1)

	ctx, cancel := context.WithCancel(ctx)

	// Done function is thread safe and will cancel the context and decrement the wait group
	// if context is already canceled, it will not do anything
	var m sync.Mutex
	done := func() {
		m.Lock()
		defer m.Unlock()
		if ctx.Err() == nil {
			cancel()
			waitGroup.Done()
		}
	}

	// After 5 seconds, if the context has not been canceled, go ahead and cancel it
	// and also decrement the wait group so that the caller can continue.
	time.AfterFunc(5*time.Second, func() {
		fmt.Fprintf(w, "Timeout waiting for port forwarding to start\n")
		done()
	})

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
					done()
				}
				if action == stopService && !service.Running && service.Port != "" {
					color.New(color.FgYellow).Fprintf(w, "Port forwarding %s:%s -> localhost stopped\n", hostname, service.Port)
					done()
				}
				return service, false
			},
		},
	)
}
