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

func GetServices(pkgs []string, rootDir string) (Services, error) {
	services := map[string]service{}
	for _, pkg := range pkgs {
		c, err := getConfigIfAny(pkg, rootDir)
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

func StartService(pkgs []string, name, rootDir string, out io.Writer) error {
	services, err := GetServices(pkgs, rootDir)
	if err != nil {
		return err
	}
	service, found := services[name]
	if !found {
		return usererr.New("Service not found")
	}
	cmd := exec.Command("sh", "-c", service.Start)
	cmd.Stdout = out
	cmd.Stderr = out
	err = cmd.Run()
	if err == nil {
		fmt.Fprintf(out, "Service %q started", name)
	}
	return usererr.WithUserMessage(err, "Service %q failed to start", name)
}

func StopService(pkgs []string, name, rootDir string, out io.Writer) error {
	services, err := GetServices(pkgs, rootDir)
	if err != nil {
		return err
	}
	service, found := services[name]
	if !found {
		return usererr.New("Service not found")
	}
	cmd := exec.Command("sh", "-c", service.Stop)
	cmd.Stdout = out
	cmd.Stderr = out
	err = cmd.Run()
	if err == nil {
		fmt.Fprintf(out, "Service %q stopped", name)
	}
	return usererr.WithUserMessage(err, "Service %q failed to stop", name)
}
