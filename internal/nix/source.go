package nix

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func SourceNixEnv() error {
	// if command is not in path, the source the nix startup files and hopefully
	// the command will be found. (we should still check that nix is actually
	// installed before we get here)
	srcFile := "/nix/var/nix/profiles/default/etc/profile.d/nix-daemon.sh"
	// if global (multi-user) daemon file is missing, try getting the single user
	// file.
	if _, err := os.Stat(srcFile); os.IsNotExist(err) {
		srcFile = filepath.Join(
			os.Getenv("HOME"),
			"/.nix-profile/etc/profile.d/nix.sh",
		)
	}

	cmd := exec.Command(
		"/bin/sh",
		"-c",
		fmt.Sprintf("source %s ; echo '<<<ENVIRONMENT>>>' ; env", srcFile),
	)

	bs, err := cmd.CombinedOutput()
	if err != nil {
		return errors.WithStack(err)
	}
	s := bufio.NewScanner(bytes.NewReader(bs))
	start := false
	for s.Scan() {
		if s.Text() == "<<<ENVIRONMENT>>>" {
			start = true
		} else if start {
			kv := strings.SplitN(s.Text(), "=", 2)
			if len(kv) == 2 {
				os.Setenv(kv[0], kv[1])
			}
		}
	}
	return nil
}
