package plugin

import (
	"encoding/json"

	"github.com/a8m/envsubst"
)

type Services map[string]service

type service struct {
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
