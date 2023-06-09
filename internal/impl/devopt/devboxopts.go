package devopt

import "io"

type Opts struct {
	Dir          string
	Pure         bool
	ShowWarnings bool
	Writer       io.Writer
}
