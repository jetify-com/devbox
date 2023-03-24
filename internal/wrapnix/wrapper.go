package wrapnix

import (
	"bytes"
	"context"
	_ "embed"
	"os"
	"path/filepath"
	"text/template"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/plugin"
)

type devboxer interface {
	NixBins(ctx context.Context) ([]string, error)
	PrintEnv() (string, error)
	ProjectDir() string
	Services() (plugin.Services, error)
}

//go:embed wrapper.sh.tmpl
var wrapper string
var wrapperTemplate = template.Must(template.New("wrapper").Parse(wrapper))

// CreateWrappers creates wrappers for all the executables in nix paths
func CreateWrappers(ctx context.Context, devbox devboxer) error {
	shellEnv, err := devbox.PrintEnv()
	if err != nil {
		return err
	}

	services, err := devbox.Services()
	if err != nil {
		return err
	}

	destPath := virtenvBinPath(devbox.ProjectDir())
	_ = os.RemoveAll(destPath)
	_ = os.MkdirAll(destPath, 0755)

	for _, service := range services {
		if err = createWrapper(&createWrapperArgs{
			Command:  service.Start,
			Env:      service.Env,
			ShellEnv: shellEnv,
			destPath: filepath.Join(destPath, service.StartName()),
		}); err != nil {
			return err
		}
		if err = createWrapper(&createWrapperArgs{
			Command:  service.Stop,
			Env:      service.Env,
			ShellEnv: shellEnv,
			destPath: filepath.Join(destPath, service.StopName()),
		}); err != nil {
			return err
		}
	}

	bins, err := devbox.NixBins(ctx)
	if err != nil {
		return err
	}

	for _, bin := range bins {
		if err = createWrapper(&createWrapperArgs{
			Command:  bin,
			ShellEnv: shellEnv,
			destPath: filepath.Join(destPath, filepath.Base(bin)),
		}); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
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
