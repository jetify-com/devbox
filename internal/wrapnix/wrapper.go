package wrapnix

import (
	"bytes"
	_ "embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/plugin"
)

type devbox interface {
	PrintEnv() (string, error)
	ProjectDir() string
	Services() (plugin.Services, error)
}

//go:embed wrapper.sh.tmpl
var wrapper string
var wrapperTemplate = template.Must(template.New("wrapper").Parse(wrapper))

// CreateWrappers creates wrappers for all the executables in the profile bin directory
// devbox struct could provide PrintEnv, but for performance, we pass it in instead
// since it's been computed already
func CreateWrappers(d devbox, shellEnv string) error {
	var err error
	if shellEnv == "" {
		shellEnv, err = d.PrintEnv()
		if err != nil {
			return err
		}
	}
	services, err := d.Services()
	if err != nil {
		return err
	}
	srcPath := profileBinPath(d.ProjectDir())
	destPath := virtenvBinPath(d.ProjectDir())
	_ = os.RemoveAll(destPath)
	_ = os.MkdirAll(destPath, 0755)

	for _, service := range services {
		if err = createWrapper(&createWrapperArgs{
			Command:  service.Start,
			Env:      service.Env,
			ShellEnv: shellEnv,
			destPath: filepath.Join(destPath, fmt.Sprintf("%s-service-start", service.Name)),
		}); err != nil {
			return err
		}
		if err = createWrapper(&createWrapperArgs{
			Command:  service.Stop,
			Env:      service.Env,
			ShellEnv: shellEnv,
			destPath: filepath.Join(destPath, fmt.Sprintf("%s-service-stop", service.Name)),
		}); err != nil {
			return err
		}
	}
	return filepath.WalkDir(
		srcPath,
		func(path string, e fs.DirEntry, err error) error {
			if err != nil {
				return errors.WithStack(err)
			}
			if !e.IsDir() {
				if err = createWrapper(&createWrapperArgs{
					Command:  path,
					ShellEnv: shellEnv,
					destPath: filepath.Join(destPath, filepath.Base(path)),
				}); err != nil {
					return errors.WithStack(err)
				}
			}
			return nil
		},
	)
}

type createWrapperArgs struct {
	Command  string
	Env      map[string]string
	ShellEnv string

	destPath string
}

func createWrapper(args *createWrapperArgs) error {
	buf := &bytes.Buffer{}
	if err := wrapperTemplate.Execute(buf, args); err != nil {
		return errors.WithStack(err)
	}

	return errors.WithStack(os.WriteFile(args.destPath, buf.Bytes(), 0755))

}

func virtenvBinPath(projectDir string) string {
	return filepath.Join(projectDir, plugin.VirtenvBinPath)
}

func profileBinPath(projectDir string) string {
	return filepath.Join(projectDir, nix.ProfileBinPath)
}
