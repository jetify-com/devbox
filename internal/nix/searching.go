package nix

import (
	"io"

	"go.jetpack.io/devbox/internal/debug"
)

func Search(writer io.Writer, url string, system string) ([]byte, error) {
	cmd := command("nix", "search", "--json", url)
	if system != "" {
		cmd.Args = append(cmd.Args, "--system", system)
	}
	cmd.Stderr = writer
	debug.Log("running command: %s\n", cmd)
	return cmd.Output()
}
