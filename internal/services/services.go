//lint:file-ignore U1000 Ignore unused function temporarily for debugging
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/a8m/envsubst"
	"github.com/fatih/color"
	"github.com/pkg/errors"

	"go.jetpack.io/devbox/internal/env"
)

type Services map[string]Service

type Service struct {
	Name               string            `json:"name"`
	Env                map[string]string `json:"-"`
	RawPort            string            `json:"port"`
	Start              string            `json:"start"`
	Stop               string            `json:"stop"`
	ProcessComposePath string
}

// TODO: (john) Since moving to process-compose, our services no longer use the old `toggleServices` function. We'll need to clean a lot of this up in a later PR.

type serviceAction int

const (
	startService serviceAction = iota
	stopService
)

func listenToAutoPortForwardingChangesOnRemote(
	ctx context.Context,
	serviceName string,
	w io.Writer,
	projectDir string,
	action serviceAction,
	cancel context.CancelFunc,
) error {
	if !env.IsCLICloudShell() {
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

func printProxyURL(w io.Writer, services Services) error { // TODO: remove it?
	if !env.IsDevboxCloud() {
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

func (s *Service) Port() (string, error) {
	if s.RawPort == "" {
		return "", nil
	}
	return envsubst.String(s.RawPort)
}

func (s *Service) ProcessComposeYaml() (string, bool) {
	return s.ProcessComposePath, true
}

func (s *Service) StartName() string {
	return fmt.Sprintf("%s-service-start", s.Name)
}

func (s *Service) StopName() string {
	return fmt.Sprintf("%s-service-stop", s.Name)
}

func (s *Services) UnmarshalJSON(b []byte) error {
	var m map[string]Service
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	*s = make(Services)
	for name, svc := range m {
		svc.Name = name
		(*s)[name] = svc
	}
	return nil
}
