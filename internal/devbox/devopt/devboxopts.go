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
	ExtraFlags         []string
	Background         bool
	ProcessComposePort int
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

type EnvrcOpts struct {
	EnvFlags
	Force     bool
	EnvrcDir  string
	ConfigDir string
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
	NoInstall             bool
	IgnoreMissingPackages bool
}

type ShellFormat string

const (
	ShellFormatBash    ShellFormat = "bash"
	ShellFormatNushell ShellFormat = "nushell"
)

type EnvExportsOpts struct {
	EnvOptions     EnvOptions
	NoRefreshAlias bool
	RunHooks       bool
	ShellFormat    ShellFormat
}

// EnvOptions configure the Devbox Environment in the `computeEnv` function.
// - These options are commonly set by flags in some Devbox commands
// like `shellenv`, `shell` and `run`.
// - The struct is designed for the "common case" to be zero-initialized as `EnvOptions{}`.
type EnvOptions struct {
	Hooks             LifecycleHooks
	OmitNixEnv        bool
	PreservePathStack bool
	Pure              bool
	SkipRecompute     bool
}

type LifecycleHooks struct {
	// OnStaleState is called when the Devbox state is out of date
	OnStaleState func()
}
