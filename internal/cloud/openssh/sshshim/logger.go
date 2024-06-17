// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package sshshim

// The sshshim is invoked by mutagen daemon, so we log errors to a file which
// we can inspect.

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/cloud/mutagenbox"
	"go.jetpack.io/devbox/internal/debug"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	logFileName = "logs.txt"
)

func EnableDebug() {
	if w, err := logFileWriter(); err == nil {
		debug.SetOutput(w)
	} else {
		fmt.Fprintf(os.Stderr, "failed to init ssh log file: %s", err)
	}
	debug.Enable()
	slog.Debug("started sshshim\n")
}

// logFile captures output for logging and when there is a failure
func logFileWriter() (io.Writer, error) {
	dirPath, err := mutagenbox.ShimDir()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &lumberjack.Logger{
		Filename:   filepath.Join(dirPath, logFileName),
		MaxSize:    2, // megabytes
		MaxBackups: 2,
		MaxAge:     28,   // days
		Compress:   true, // disabled by default
	}, nil
}
