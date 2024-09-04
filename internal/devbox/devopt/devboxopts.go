package devopt

import (
	"io"
)

// Naming Convention:
// - suffix Opts for structs corresponding to a Devbox api function
// - omit suffix Opts for other structs that are composed into an Opts struct

type Opts struct {
	Dir                      string
	Env                      map[string]string
	Environment              string
	IgnoreWarnings           bool
	CustomProcessComposeFile string
	Stderr                   io.Writer
}

type ProcessComposeOpts struct {
	ExtraFlags []string
	Background bool
}

type GenerateOpts struct {
	ForType  string
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

type AddOpts struct {
	AllowInsecure    []string
	Platforms        []string
	ExcludePlatforms []string
	DisablePlugin    bool
	Patch            string
	Outputs          []string
}

type UpdateOpts struct {
	Pkgs                  []string
	IgnoreMissingPackages bool
}

type EnvExportsOpts struct {
	DontRecomputeEnvironment bool
	EnvOptions               EnvOptions
	NoRefreshAlias           bool
	RunHooks                 bool
}

// EnvOptions configure the Devbox Environment in the `computeEnv` function.
// - These options are commonly set by flags in some Devbox commands
// like `shellenv`, `shell` and `run`.
// - The struct is designed for the "common case" to be zero-initialized as `EnvOptions{}`.
type EnvOptions struct {
	OmitNixEnv        bool
	PreservePathStack bool
	Pure              bool
}
