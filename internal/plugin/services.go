package plugin

import (
	"go.jetpack.io/devbox/internal/services"
)

func GetServices(pkgs, includes []string, projectDir string) (services.Services, error) {
	svcs := services.Services{}

	allPkgs := append([]string(nil), pkgs...)
	for _, include := range includes {
		name, err := parseInclude(include)
		if err != nil {
			return nil, err
		}
		allPkgs = append(allPkgs, name)
	}

	for _, pkg := range allPkgs {
		conf, err := getConfigIfAny(pkg, projectDir)
		if err != nil {
			return nil, err
		}
		if conf == nil {
			continue
		}

		if file, hasProcessComposeYaml := conf.ProcessComposeYaml(); hasProcessComposeYaml {
			svc := services.Service{
				Name:               conf.Name,
				Env:                conf.Env,
				ProcessComposePath: file,
			}
			svcs[conf.Name] = svc
		}

	}
	return svcs, nil
}
