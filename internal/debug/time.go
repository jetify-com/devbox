// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package debug

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var timerEnabled, _ = strconv.ParseBool(os.Getenv(devboxPrintExecTime))

const devboxPrintExecTime = "DEVBOX_PRINT_EXEC_TIME"

var headerPrinted = false

type timer struct {
	name string
	time time.Time
}

func Timer(name string) *timer {
	if !timerEnabled {
		return nil
	}
	return &timer{
		name: name,
		time: time.Now(),
	}
}

func FunctionTimer() *timer {
	if !timerEnabled {
		return nil
	}
	pc := make([]uintptr, 15)
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	parts := strings.Split(frame.Function, ".")
	return Timer(parts[len(parts)-1])
}

func (t *timer) End() {
	if t == nil {
		return
	}
	if !headerPrinted {
		fmt.Fprintln(os.Stderr, "\nExec times over 1ms:")
		headerPrinted = true
	}
	if time.Since(t.time) >= time.Millisecond {
		fmt.Fprintf(os.Stderr, "\"%s\" took %s\n", t.name, time.Since(t.time))
	}
}
