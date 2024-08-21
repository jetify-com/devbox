// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package ux

import (
	"context"
	"fmt"
	"io"

	"github.com/fatih/color"
)

var (
	success = color.New(color.FgHiGreen)
	info    = color.New(color.FgYellow)
	warning = color.New(color.FgHiYellow)
	error   = color.New(color.FgHiRed)
)

func Fsuccess(w io.Writer, a ...any) {
	success.Fprint(w, "Success: ")
	fmt.Fprint(w, a...)
}

func Fsuccessf(w io.Writer, format string, a ...any) {
	success.Fprint(w, "Success: ")
	fmt.Fprintf(w, format, a...)
}

func Finfo(w io.Writer, a ...any) {
	info.Fprint(w, "Info: ")
	fmt.Fprint(w, a...)
}

func Finfof(w io.Writer, format string, a ...any) {
	info.Fprint(w, "Info: ")
	fmt.Fprintf(w, format, a...)
}

func Fwarning(w io.Writer, a ...any) {
	warning.Fprint(w, "Warning: ")
	fmt.Fprint(w, a...)
}

func Fwarningf(w io.Writer, format string, a ...any) {
	warning.Fprint(w, "Warning: ")
	fmt.Fprintf(w, format, a...)
}

func Ferror(w io.Writer, a ...any) {
	error.Fprint(w, "Error: ")
	fmt.Fprint(w, a...)
}

func Ferrorf(w io.Writer, format string, a ...any) {
	error.Fprint(w, "Error: ")
	fmt.Fprintf(w, format, a...)
}

// Hidable messages allow the use of context to disable a message. Messages can be hidden
// by their format string.

type ctxKey string

func HideMessage(ctx context.Context, format string) context.Context {
	return context.WithValue(ctx, ctxKey(format), true)
}

func FHidableWarning(ctx context.Context, w io.Writer, format string, a ...any) {
	if isHidden(ctx, format) {
		return
	}
	Fwarningf(w, format, a...)
}

func isHidden(ctx context.Context, format string) bool {
	isHidden, _ := ctx.Value(ctxKey(format)).(bool)
	return isHidden
}
