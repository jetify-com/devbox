package shenv

type unknown struct{}

// UnknownSh adds support the unknown shell. This serves
// as a fallback alternative to outright failure.
var UnknownSh Shell = unknown{}

const unknownHook = `
echo "Warning: this shell will not update its environment. 
Please exit and re-enter shell after making any changes that may affect the devbox generated environment.\n"
`

func (sh unknown) Hook() (string, error) {
	return unknownHook, nil
}

func (sh unknown) Export(e ShellExport) (out string) {
	panic("not implemented")
}

func (sh unknown) Dump(env Env) (out string) {
	panic("not implemented")
}
