package sshshim

// The sshshim is invoked by mutagen daemon, so we log errors to a file which
// we can inspect.

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/debug"
)

const (
	logFileName = "ssh.log"
)

func EnableDebug() {
	if w, err := logFile(); err == nil {
		debug.SetOutput(w)
	} else {
		fmt.Fprintf(os.Stderr, "failed to init ssh log file: %s", err)
	}
	debug.Enable()
	debug.Log("started sshshim\n")
}

// logFile captures output for logging and when there is a failure
// NOTE: Ideally, we should limit the size of this log file, but it is always truncated
// because only the last ssh invocation (which may have failed) has its output saved.
// So size should hopefully not be crazy big.
func logFile() (io.Writer, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	dirPath := filepath.Join(home, configShimDir)

	file, err := os.OpenFile(
		filepath.Join(dirPath, logFileName),
		os.O_RDWR|os.O_CREATE|os.O_TRUNC,
		0700,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return file, nil
}
