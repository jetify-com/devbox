package nix

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/hashicorp/go-envparse"
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

	// Source the nix script that sets the environment, and print the environment
	// variables that it sets, in a way we can parse.

	// NOTE: Only use shell built-ins in this script so that we don't introduce
	// any dependencies on an external binary.
	script := heredoc.Docf(`
		. %s;
		echo PATH=$PATH;
		echo NIX_PROFILES=$NIX_PROFILES;
		echo NIX_SSL_CERT_FILE=$NIX_SSL_CERT_FILE;
		echo MANPATH=$MANPATH;
	`, srcFile)
	cmd := exec.Command(
		"/bin/sh",
		"-c",
		script,
	)

	bs, err := cmd.CombinedOutput()
	if err != nil {
		// When there's an error, the output is usually an error message that
		// was printed to stderr and that we want in the error for debugging.
		return errors.Wrap(err, string(bs))
	}
	envvars, err := envparse.Parse(bytes.NewReader(bs))

	if err != nil {
		return errors.Wrap(err, "failed to parse nix env vars")
	}

	for k, v := range envvars {
		if len(strings.TrimSpace(v)) > 0 {
			os.Setenv(k, v)
		}
	}

	return nil
}
