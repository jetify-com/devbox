package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
)

type Services map[string]service

type service struct {
	Name  string `json:"name"`
	Start string `json:"start"`
	Stop  string `json:"stop"`
}

func GetServices(pkgs []string, projectDir string) (Services, error) {
	services := map[string]service{}
	for _, pkg := range pkgs {
		c, err := getConfigIfAny(pkg, projectDir)
		if err != nil {
			return nil, err
		}
		if c == nil {
			continue
		}
		for name, svc := range c.Services {
			svc.Name = name
			services[name] = svc
		}
	}
	return services, nil
}

func (s *Services) UnmarshalJSON(b []byte) error {
	var m map[string]service
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

func StartServices(pkgs, serviceNames []string, root string, w io.Writer) error {
	services, err := GetServices(pkgs, root)
	if err != nil {
		return err
	}
	for _, name := range serviceNames {
		service, found := services[name]
		if !found {
			return usererr.New("Service not found")
		}
		cmd := exec.Command("sh", "-c", service.Start)
		cmd.Stdout = w
		cmd.Stderr = w
		if err = cmd.Run(); err != nil {
			if len(serviceNames) == 1 {
				return usererr.WithUserMessage(err, "Service %q failed to start", name)
			}
			fmt.Fprintf(w, "Service %q failed to start. Error = %s\n", name, err)
		} else {
			fmt.Fprintf(w, "Service %q started\n", name)
		}
	}
	return nil
}

func StopServices(pkgs, serviceNames []string, root string, w io.Writer) error {
	services, err := GetServices(pkgs, root)
	if err != nil {
		return err
	}
	for _, name := range serviceNames {
		service, found := services[name]
		if !found {
			return usererr.New("Service not found")
		}
		cmd := exec.Command("sh", "-c", service.Stop)
		cmd.Stdout = w
		cmd.Stderr = w
		if err = cmd.Run(); err != nil {
			if len(serviceNames) == 1 {
				return usererr.WithUserMessage(err, "Service %q failed to stop", name)
			}
			fmt.Fprintf(w, "Service %q failed to stop. Error = %s\n", name, err)
		} else {
			fmt.Fprintf(w, "Service %q stopped\n", name)
		}
	}
	return nil
}
