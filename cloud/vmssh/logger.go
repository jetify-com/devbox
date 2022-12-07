package vmssh

// The vmssh shim is invoked by mutagen daemon, so we log errors to a file which
// we can inspect.

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

const (
	logDir      = ".config/devbox/log"
	logFileName = "devbox_cloud_ssh.log"
)

var logger *sshLogger = NewSSHLogger()

type sshLogger struct {
	writer io.Writer
}

func NewSSHLogger() *sshLogger {
	w, err := logFile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init ssh log file: %s", err)
		return nil
	}

	return &sshLogger{
		writer: w,
	}
}

func (l *sshLogger) log(msg string, args ...any) {
	if l == nil {
		return
	}
	fmt.Fprintf(l.writer, msg, args...)
}

// logFile captures output when there is a failure
// NOTE: we should limit the size of this log file.
func logFile() (io.Writer, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	dirPath := filepath.Join(home, logDir)
	if err = os.MkdirAll(dirPath, 0700); err != nil {
		return nil, errors.WithStack(err)
	}

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
