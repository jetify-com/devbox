package devopt

import (
	"io"
)

type Opts struct {
	AllowInsecureAdds        bool
	Dir                      string
	Env                      map[string]string
	Pure                     bool
	IgnoreWarnings           bool
	CustomProcessComposeFile string
	OmitBinWrappersFromPath  bool
	Writer                   io.Writer
}

type GenerateOpts struct {
	Force    bool
	RootUser bool
}

type EnvFlags struct {
	EnvMap  map[string]string
	EnvFile string
}
