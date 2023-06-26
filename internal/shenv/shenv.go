package shenv

type Env map[string]string

// Shell is the interface that represents the interaction with the host shell.
type Shell interface {
	// Hook is the string that gets evaluated into the host shell config and
	// setups direnv as a prompt hook.
	Hook() (string, error)

	// Export outputs the ShellExport as an evaluatable string on the host shell
	Export(e ShellExport) string

	// Dump outputs and evaluatable string that sets the env in the host shell
	Dump(env Env) string
}

// ShellExport represents environment variables to add and remove on the host
// shell.
type ShellExport map[string]*string

// Add represents the addition of a new environment variable
func (e ShellExport) Add(key, value string) {
	e[key] = &value
}

// Remove represents the removal of a given `key` environment variable.
func (e ShellExport) Remove(key string) {
	e[key] = nil
}

// DetectShell returns a Shell instance from the given shell name
// TODO: use a single common "enum" for both shenv and DevboxShell
func DetectShell(target string) Shell {
	switch target {
	case "bash":
		return Bash
	case "fish":
		return Fish
	case "ksh":
		return Ksh
	case "posix":
		return Posix
	case "zsh":
		return Zsh
	default:
		return UnknownSh
	}
}
