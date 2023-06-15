package shenv

// ZSH is a singleton instance of ZSH_T
type zsh struct{}

// Zsh adds support for the venerable Z shell.
var Zsh Shell = zsh{}

const zshHook = `
_devbox_hook() {
  trap -- '' SIGINT;
  eval "$(devbox shellenv --config {{ .ProjectDir }})";
  trap - SIGINT;
}
typeset -ag precmd_functions;
if [[ -z "${precmd_functions[(r)_devbox_hook]+1}" ]]; then
  precmd_functions=( _devbox_hook ${precmd_functions[@]} )
fi
`

func (sh zsh) Hook() (string, error) {
	return zshHook, nil
}

func (sh zsh) Export(e ShellExport) (out string) {
	for key, value := range e {
		if value == nil {
			out += sh.unset(key)
		} else {
			out += sh.export(key, *value)
		}
	}
	return out
}

func (sh zsh) Dump(env Env) (out string) {
	for key, value := range env {
		out += sh.export(key, value)
	}
	return out
}

func (sh zsh) export(key, value string) string {
	return "export " + sh.escape(key) + "=" + sh.escape(value) + ";"
}

func (sh zsh) unset(key string) string {
	return "unset " + sh.escape(key) + ";"
}

func (sh zsh) escape(str string) string {
	return BashEscape(str)
}
