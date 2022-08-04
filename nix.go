package devbox

import (
	"os"
	"os/exec"
	"strings"
)

func Shell(path string) {
	cfg := LoadDevConfig(path)
	err := Generate(path, cfg)
	if err != nil {
		panic(err)
	}
	cmd := exec.Command("nix-shell")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = path
	_ = cmd.Run()
}

func Exec(path string, args []string) {
	runCmd := strings.Join(args, " ")
	cmd := exec.Command("nix-shell", "--run", runCmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = path
	_ = cmd.Run()
}
