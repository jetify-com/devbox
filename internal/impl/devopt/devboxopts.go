package devopt

import (
	"io"
)

type Opts struct {
	Dir            string
	Pure           bool
	IgnoreWarnings bool
	Writer         io.Writer
}
