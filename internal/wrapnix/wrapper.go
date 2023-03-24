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
	ProjectDir() string
	Services() (plugin.Services, error)
}

//go:embed wrapper.sh.tmpl
var wrapper string
var wrapperTemplate = template.Must(template.New("wrapper").Parse(wrapper))

func CreateWrappers(d devbox) error {
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
			destPath: filepath.Join(destPath, fmt.Sprintf("%s-service-start", service.Name)),
			Env:      service.Env,
		}); err != nil {
			return err
		}
		if err = createWrapper(&createWrapperArgs{
			Command:  service.Stop,
			destPath: filepath.Join(destPath, fmt.Sprintf("%s-service-stop", service.Name)),
			Env:      service.Env,
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
	destPath string
	Env      map[string]string
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
