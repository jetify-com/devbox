package devopt

import (
	"io"
)

type Opts struct {
	AllowInsecureAdds        bool
	Dir                      string
	Pure                     bool
	IgnoreWarnings           bool
	CustomProcessComposeFile string
	Writer                   io.Writer
}
