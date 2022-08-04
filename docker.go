package devbox

import (
	"os"
	"os/exec"
)

func Build(path string) {
	cmd := exec.Command("docker", "build", ".")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "BUILDKIT=1")
	cmd.Dir = path
	_ = cmd.Run()
}
