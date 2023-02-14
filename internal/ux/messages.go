package ux

import (
	"fmt"
	"io"

	"github.com/fatih/color"
)

func Fwarning(w io.Writer, format string, a ...any) {
	color.New(color.FgHiYellow).Fprint(w, "Warning: ")
	fmt.Fprintf(w, format, a...)
}
