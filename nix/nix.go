package nix

import (
	"os"
	"os/exec"
	"strings"
)

func Shell(path string) error {
	cmd := exec.Command("nix-shell")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = path
	return cmd.Run()
}

func Exec(path string, command []string) error {
	runCmd := strings.Join(command, " ")
	cmd := exec.Command("nix-shell", "--run", runCmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = path
	return cmd.Run()
}
