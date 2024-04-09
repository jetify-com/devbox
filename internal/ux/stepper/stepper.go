// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package stepper

import (
	"fmt"
	"io"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

type Stepper struct {
	spinner *spinner.Spinner
}

func Start(w io.Writer, format string, a ...any) *Stepper {
	spinner := spinner.New(spinner.CharSets[11], 100*time.Millisecond, spinner.WithWriter(w))
	err := spinner.Color("magenta")
	if err != nil {
		panic(err)
	}
	spinner.Suffix = " " + fmt.Sprintf(format, a...)
	spinner.Start()
	return &Stepper{
		spinner: spinner,
	}
}

func (s *Stepper) Stop(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	s.spinner.FinalMSG = fmt.Sprintf("%s %s\n", color.BlueString("→"), msg)
	s.spinner.Stop()
}

func (s *Stepper) Fail(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	s.spinner.FinalMSG = fmt.Sprintf("%s %s\n", color.RedString("✘"), msg)
	s.spinner.Stop()
}

func (s *Stepper) Success(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	s.spinner.FinalMSG = fmt.Sprintf("%s %s\n", color.GreenString("✓"), msg)
	s.spinner.Stop()
}

func (s *Stepper) Display(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	// we need to add a space prefix to give a small gap between the spinner animation and the msg
	s.spinner.Suffix = fmt.Sprintf(" %s", msg)
}
