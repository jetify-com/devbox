package plugin

import (
	"go.jetpack.io/devbox/internal/services"
)

func GetServices(pkgs []string, projectDir string) (services.Services, error) {
	svcs := services.Services{}
	for _, pkg := range pkgs {
		c, err := getConfigIfAny(pkg, projectDir)
		if err != nil {
			return nil, err
		}
		if c == nil {
			continue
		}

		if file, hasProcessComposeYaml := c.ProcessComposeYaml(); hasProcessComposeYaml {
			svc := services.Service{
				Name:               c.Name,
				Env:                c.Env,
				ProcessComposePath: file,
			}
			svcs[c.Name] = svc
		}

	}
	return svcs, nil
}
