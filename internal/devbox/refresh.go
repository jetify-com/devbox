package devbox

import (
	"fmt"
	"os"
	"strings"
)

func (d *Devbox) IsDirenvActive() bool {
	return strings.TrimPrefix(os.Getenv("DIRENV_DIR"), "-") == d.projectDir
}

func (d *Devbox) isRefreshAliasSet() bool {
	return os.Getenv(d.refreshAliasEnvVar()) == d.refreshCmd()
}

func (d *Devbox) refreshAliasEnvVar() string {
	return "DEVBOX_REFRESH_ALIAS_" + d.ProjectDirHash()
}

func (d *Devbox) isGlobal() bool {
	globalPath, _ := GlobalDataPath()
	return d.projectDir == globalPath
}

// In some cases (e.g. 2 non-global projects somehow active at the same time),
// refresh might not match. This is a tiny edge case, so no need to make UX
// great, we just print out the entire command.
func (d *Devbox) RefreshAliasOrCommand() string {
	if !d.isRefreshAliasSet() {
		// even if alias is not set, it might still be set by the end of this process
		return fmt.Sprintf("`%s` or `%s`", d.refreshAliasName(), d.refreshCmd())
	}
	return d.refreshAliasName()
}

func (d *Devbox) refreshAliasName() string {
	if d.isGlobal() {
		return "refresh-global"
	}
	return "refresh"
}

func (d *Devbox) refreshCmd() string {
	devboxCmd := fmt.Sprintf("shellenv --preserve-path-stack -c %q", d.projectDir)
	if d.isGlobal() {
		devboxCmd = "global shellenv --preserve-path-stack -r"
	}
	if isFishShell() {
		return fmt.Sprintf(`eval (devbox %s  | string collect)`, devboxCmd)
	}
	return fmt.Sprintf(`eval "$(devbox %s)" && hash -r`, devboxCmd)
}

func (d *Devbox) refreshCmdForShell(format string) string {
	devboxCmd := fmt.Sprintf("shellenv --preserve-path-stack -c %q", d.projectDir)
	if d.isGlobal() {
		devboxCmd = "global shellenv --preserve-path-stack -r --format " + format
	} else {
		devboxCmd = fmt.Sprintf("shellenv --preserve-path-stack -c %q --format %s", d.projectDir, format)
	}

	if format == "nushell" {
		// Nushell doesn't have eval; use overlay or source with temporary file
		return fmt.Sprintf(`devbox %s | save -f ~/.cache/devbox-env.nu; source ~/.cache/devbox-env.nu`, devboxCmd)
	}
	if format == "fish" || isFishShell() {
		return fmt.Sprintf(`eval (devbox %s  | string collect)`, devboxCmd)
	}
	return fmt.Sprintf(`eval "$(devbox %s)" && hash -r`, devboxCmd)
}

func (d *Devbox) refreshAlias() string {
	if isFishShell() {
		return fmt.Sprintf(
			`if not type %[1]s >/dev/null 2>&1
	export %[2]s='%[3]s'
	alias %[1]s='%[3]s'
end`,
			d.refreshAliasName(),
			d.refreshAliasEnvVar(),
			d.refreshCmd(),
		)
	}
	return fmt.Sprintf(
		`if ! type %[1]s >/dev/null 2>&1; then
	export %[2]s='%[3]s'
	alias %[1]s='%[3]s'
fi`,
		d.refreshAliasName(),
		d.refreshAliasEnvVar(),
		d.refreshCmd(),
	)
}

func (d *Devbox) refreshAliasForShell(format string) string {
	// For nushell format, provide instructions as a comment since aliases with pipes are complex
	if format == "nushell" {
		devboxCmd := "global shellenv --preserve-path-stack -r --format nushell"
		if !d.isGlobal() {
			devboxCmd = fmt.Sprintf("shellenv --preserve-path-stack -c %q --format nushell", d.projectDir)
		}
		return fmt.Sprintf(
			`# To refresh your devbox environment in nushell, run:
# devbox %s | save -f ~/.cache/devbox-env.nu; source ~/.cache/devbox-env.nu`,
			devboxCmd,
		)
	}
	// Otherwise use the original refreshAlias function
	return d.refreshAlias()
}
