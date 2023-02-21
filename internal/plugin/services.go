package plugin

import (
	"encoding/json"
	"strings"

	"github.com/a8m/envsubst"
)

type Services map[string]service

type service struct {
	config  *config
	Name    string `json:"name"`
	RawPort string `json:"port"`
	Start   string `json:"start"`
	Stop    string `json:"stop"`
}

func (s *service) Port() (string, error) {
	if s.RawPort == "" {
		return "", nil
	}
	return envsubst.String(s.RawPort)
}

func (s *service) ProcessComposeYaml() (string, bool) {
	for file := range s.config.CreateFiles {
		if strings.HasSuffix(file, "process-compose.yaml") || strings.HasSuffix(file, "process-compose.yml") {
			return file, true
		}
	}
	return "", false
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
			svc.config = c
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
