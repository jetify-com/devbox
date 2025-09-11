package nix

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
)

// Installer downloads and installs Nix.
type Installer struct {
	// Path is the path to the Nix installer. If it's empty, Download and
	// Install will automatically download the installer and set Path to the
	// downloaded file before returning.
	Path string
}

// Download downloads the Nix installer without running it.
func (i *Installer) Download(ctx context.Context) error {
	if i.Path != "" {
		return fmt.Errorf("installer already downloaded: %s", i.Path)
	}

	system := ""
	switch runtime.GOARCH {
	case "amd64":
		switch runtime.GOOS {
		case "darwin":
			system = "x86_64-darwin"
		case "linux":
			system = "x86_64-linux"
		}
	case "arm64":
		switch runtime.GOOS {
		case "darwin":
			system = "aarch64-darwin"
		case "linux":
			system = "aarch64-linux"
		}
	}

	url := "https://github.com/NixOS/experimental-nix-installer/releases/download/0.27.0/nix-installer-" + system
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %s", resp.Status)
	}
	installer, err := writeTempFile(resp.Body)
	if err != nil {
		return err
	}
	err = os.Chmod(installer, 0o755)
	if err != nil {
		return fmt.Errorf("chmod 0755 installer: %v", err)
	}
	i.Path = installer
	return nil
}

// Run downloads and installs Nix.
func (i *Installer) Run(ctx context.Context) error {
	if i.Path == "" {
		err := i.Download(ctx)
		if err != nil {
			return err
		}
	}

	cmd := exec.CommandContext(ctx, i.Path, "install")
	switch runtime.GOOS {
	case "darwin":
		cmd.Args = append(cmd.Args, "macos")
	case "linux":
		cmd.Args = append(cmd.Args, "linux")
		_, err := os.Stat("/run/systemd/system")
		if errors.Is(err, os.ErrNotExist) {
			// Respect any env var settings from the user.
			_, ok := os.LookupEnv("NIX_INSTALLER_INIT")
			if !ok {
				cmd.Args = append(cmd.Args, "--init", "none")
			}
		}
	}
	cmd.Args = append(cmd.Args, "--no-confirm")
	cmd.Cancel = func() error {
		return cmd.Process.Signal(os.Interrupt)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run installer: %v", err)
	}
	_, _ = SourceProfile()
	return nil
}

func writeTempFile(r io.Reader) (path string, err error) {
	tempFile, err := os.CreateTemp("", "devbox-nix-installer-")
	if err != nil {
		return "", fmt.Errorf("create temp file: %v", err)
	}

	_, err = io.Copy(tempFile, r)
	closeErr := tempFile.Close()
	if err == nil && closeErr != nil {
		err = fmt.Errorf("close temp file: %v", closeErr)
	}

	if err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}
	return tempFile.Name(), nil
}
