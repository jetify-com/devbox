package services

import (
	"context"
	"io"
	"os/exec"
)

func StartServices(ctx context.Context, w io.Writer, serviceNames []string, projectDir string) error {
	return clientRequest("start", serviceNames)
}

func StopServices(ctx context.Context, pkgs, serviceNames []string, projectDir string, w io.Writer) error {
	return clientRequest("stop", serviceNames)
}

func RestartServices(ctx context.Context, pkgs, serviceNames []string, projectDir string, w io.Writer) error {
	return clientRequest("restart", serviceNames)
}

func ListServices(ctx context.Context, pkgs, serviceNames []string, projectDir string, w io.Writer) error {
	return clientRequest("list", serviceNames)
}

func clientRequest(subCmd string, serviceNames []string) error {
	flags := []string{"-p", "8280"}
	listCommand := []string{"process", subCmd}

	if len(serviceNames) > 0 {
		flags = append(serviceNames, flags...)
		flags = append(listCommand, flags...)
	}

	cmd := exec.Command("process-compose", flags...)
	return cmd.Run()
}
