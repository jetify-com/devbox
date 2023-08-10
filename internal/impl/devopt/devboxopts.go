package devopt

import (
	"io"
)

type GenerateOpts struct {
	Force    bool
	RootUser bool
}

type Opts struct {
	AllowInsecureAdds        bool
	Dir                      string
	Env                      map[string]string
	Pure                     bool
	IgnoreWarnings           bool
	CustomProcessComposeFile string
	Writer                   io.Writer
	GenerateOpts             GenerateOpts
}

type EnvFlags struct {
	EnvMap  map[string]string
	EnvFile string
}
