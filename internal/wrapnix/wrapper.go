package wrapnix

import (
	"bytes"
	"context"
	_ "embed"
	"os"
	"path/filepath"
	"text/template"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/nix"
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

	// Remove all old wrappers
	_ = os.RemoveAll(filepath.Join(devbox.ProjectDir(), plugin.WrapperPath))

	// Recreate the bin wrapper directory
	destPath := filepath.Join(devbox.ProjectDir(), plugin.WrapperBinPath)
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

	return createSymlinksForSupportDirs(devbox.ProjectDir())
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

// createSymlinksForSupportDirs creates symlinks for the support dirs
// (etc, lib, share) in the virtenv. Some tools (like mariadb) expect
// these to be in a dir relative to the bin.
//
// TODO: this is not perfect. using the profile path will not take into account
// any special stuff we do in flake.nix. We should use the nix store directly,
// but that is a bit more complicated. Nix merges any support directories
// recursively, so we need to do the same.
// e.g. if go_1_19 and go_1_20 are installed, .devbox/nix/profile/default/share/go/api
// will contain the union of both. We need to do the same.
func createSymlinksForSupportDirs(projectDir string) error {
	profilePath := filepath.Join(projectDir, nix.ProfilePath)
	if _, err := os.Stat(profilePath); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	supportDirs, err := os.ReadDir(profilePath)
	if err != nil {
		return err
	}

	for _, dir := range supportDirs {
		// bin has wrappers and is not a symlink
		if dir.Name() == "bin" {
			continue
		}

		oldname := filepath.Join(projectDir, nix.ProfilePath, dir.Name())
		newname := filepath.Join(projectDir, plugin.WrapperPath, dir.Name())

		if err := os.Symlink(oldname, newname); err != nil {
			return err
		}
	}
	return nil
}
