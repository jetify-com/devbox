package shenv

type ksh struct{}

// Ksh adds support the korn shell
var Ksh Shell = ksh{}

// um, this is ChatGPT writing it. I need to verify and test
const kshHook = `
_devbox_hook() {
  eval "$(devbox shellenv --config {{ .ProjectDir }})";
}
if [[ "$(typeset -f precmd)" != *"_devbox_hook"* ]]; then
  function precmd {
    devbox_hook
  }
fi
`

func (sh ksh) Hook() (string, error) {
	return kshHook, nil
}

func (sh ksh) Export(e ShellExport) (out string) {
	panic("not implemented")
}

func (sh ksh) Dump(env Env) (out string) {
	panic("not implemented")
}
