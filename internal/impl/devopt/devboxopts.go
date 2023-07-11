package devopt

import (
	"io"
)

type Opts struct {
	AllowInsecureAdds bool
	Dir               string
	Pure              bool
	IgnoreWarnings    bool
	Writer            io.Writer
}
