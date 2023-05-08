// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package ux

import (
	"fmt"
	"io"

	"github.com/fatih/color"
)

func Fsuccess(w io.Writer, format string, a ...any) {
	color.New(color.FgHiGreen).Fprint(w, "Success: ")
	fmt.Fprintf(w, format, a...)
}

func Fwarning(w io.Writer, format string, a ...any) {
	color.New(color.FgHiYellow).Fprint(w, "Warning: ")
	fmt.Fprintf(w, format, a...)
}

func Ferror(w io.Writer, format string, a ...any) {
	color.New(color.FgHiRed).Fprint(w, "Error: ")
	fmt.Fprintf(w, format, a...)
}
