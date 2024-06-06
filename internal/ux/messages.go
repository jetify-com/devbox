// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package ux

import (
	"context"
	"fmt"
	"io"

	"github.com/fatih/color"
)

func Fsuccess(w io.Writer, format string, a ...any) {
	color.New(color.FgHiGreen).Fprint(w, "Success: ")
	fmt.Fprintf(w, format, a...)
}

func Finfo(w io.Writer, format string, a ...any) {
	color.New(color.FgYellow).Fprint(w, "Info: ")
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
	Fwarning(w, format, a...)
}

func isHidden(ctx context.Context, format string) bool {
	isHidden, _ := ctx.Value(ctxKey(format)).(bool)
	return isHidden
}
