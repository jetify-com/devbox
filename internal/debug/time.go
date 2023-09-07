// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package debug

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const devboxPrintExecTime = "DEVBOX_PRINT_EXEC_TIME"

var start = time.Now()

func PrintExecutionTime() {
	if enabled, _ := strconv.ParseBool(os.Getenv(devboxPrintExecTime)); !enabled {
		return
	}
	fmt.Fprintf(
		os.Stderr,
		"\"%s\" took %s\n", strings.Join(os.Args, " "),
		time.Since(start),
	)
}
