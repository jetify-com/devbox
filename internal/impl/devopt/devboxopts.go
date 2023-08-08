package devopt

import (
	"io"
)

type Opts struct {
	AllowInsecureAdds bool
	Dir               string
	Env               map[string]string
	Pure              bool
	IgnoreWarnings    bool
	Writer            io.Writer
}
