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

type PullboxOpts struct {
	Overwrite   bool
	URL         string
	Credentials Credentials
}

type Credentials struct {
	IDToken string
	// TODO We can just parse these out, but don't want to add a dependency right now
	Email string
	Sub   string
}

type UpdateOpts struct {
	Pkgs []string
	Sync bool
}
