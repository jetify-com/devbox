// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"bytes"
	_ "embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/alessio/shellescape"
	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/debug"
)

//go:embed shellrc.tmpl
var shellrcText string
var shellrcTmpl = template.Must(template.New("shellrc").Parse(shellrcText))

type name string

const (
	shUnknown name = ""
	shBash    name = "bash"
	shZsh     name = "zsh"
	shKsh     name = "ksh"
	shPosix   name = "posix"
)

// Shell configures a user's shell to run in Devbox. Its zero value is a
// fallback shell that launches a regular Nix shell.
type Shell struct {
	name            name
	binPath         string
	projectDir      string // path to where devbox.json config resides
	pkgConfigDir    string
	env             []string
	userShellrcPath string
	pluginInitHook  string

	// UserInitHook contains commands that will run at shell startup.
	UserInitHook string

	ScriptName    string
	ScriptCommand string

	// profileDir is the absolute path to the directory storing the nix-profile
	profileDir  string
	historyFile string

	// shellStartTime is the unix timestamp for when the command was invoked
	shellStartTime string
}

type ShellOption func(*Shell)

// DetectShell attempts to determine the user's default shell.
func DetectShell(opts ...ShellOption) (*Shell, error) {
	path := os.Getenv("SHELL")
	if path == "" {
		return nil, errors.New("unable to detect the current shell")
	}

	sh := &Shell{binPath: filepath.Clean(path)}
	base := filepath.Base(path)
	// Login shell
	if base[0] == '-' {
		base = base[1:]
	}
	switch base {
	case "bash":
		sh.name = shBash
		sh.userShellrcPath = rcfilePath(".bashrc")
	case "zsh":
		sh.name = shZsh
		sh.userShellrcPath = rcfilePath(".zshrc")
	case "ksh":
		sh.name = shKsh
		sh.userShellrcPath = rcfilePath(".kshrc")
	case "dash", "ash", "sh":
		sh.name = shPosix
		sh.userShellrcPath = os.Getenv("ENV")

		// Just make up a name if there isn't already an init file set
		// so we have somewhere to put a new one.
		if sh.userShellrcPath == "" {
			sh.userShellrcPath = ".shinit"
		}
	default:
		sh.name = shUnknown
	}

	for _, opt := range opts {
		opt(sh)
	}

	debug.Log("Recognized shell as: %s", sh.binPath)
	debug.Log("Looking for user's shell init file at: %s", sh.userShellrcPath)
	return sh, nil
}

// If/once we end up making plugins the same as devbox.json we probably want
// to merge all init hooks into single field
func WithPluginInitHook(hook string) ShellOption {
	return func(s *Shell) {
		s.pluginInitHook = hook
	}
}

func WithProfile(profileDir string) ShellOption {
	return func(s *Shell) {
		s.profileDir = profileDir
	}
}

func WithHistoryFile(historyFile string) ShellOption {
	return func(s *Shell) {
		s.historyFile = historyFile
	}
}

// TODO: Consider removing this once plugins add env vars directly to binaries
// via wrapper scripts.
func WithEnvVariables(envVariables map[string]string) ShellOption {
	return func(s *Shell) {
		for k, v := range envVariables {
			s.env = append(s.env, fmt.Sprintf("%s=%s", k, v))
		}
	}
}

func WithUserScript(name string, command string) ShellOption {
	return func(s *Shell) {
		s.ScriptName = name
		s.ScriptCommand = command
	}
}

func WithPKGConfigDir(pkgConfigDir string) ShellOption {
	return func(s *Shell) {
		s.pkgConfigDir = pkgConfigDir
	}
}

func WithProjectDir(projectDir string) ShellOption {
	return func(s *Shell) {
		s.projectDir = projectDir
	}
}

func WithShellStartTime(time string) ShellOption {
	return func(s *Shell) {
		s.shellStartTime = time
	}
}

// rcfilePath returns the absolute path for an rcfile, which is usually in the
// user's home directory. It doesn't guarantee that the file exists.
func rcfilePath(basename string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, basename)
}

func (s *Shell) Run(nixShellFilePath, nixFlakesFilePath string) error {
	// Copy the current PATH into nix-shell, but clean and remove some
	// directories that are incompatible.
	parentPath := CleanEnvPath(os.Getenv("PATH"), os.Getenv("NIX_PROFILES"))

	env := append(s.env, os.Environ()...)
	env = append(
		env,
		"PARENT_PATH="+parentPath,

		// Prevent the user's shellrc from re-sourcing nix-daemon.sh
		// inside the devbox shell.
		"__ETC_PROFILE_NIX_SOURCED=1",
		// Always allow unfree packages.
		"NIXPKGS_ALLOW_UNFREE=1",
	)

	// Launch a fallback shell if we couldn't find the path to the user's
	// default shell.
	if s.binPath == "" {
		if featureflag.Flakes.Enabled() {
			return errors.New("No default shell not supported in Flakes mode")
		}
		cmd := exec.Command("nix-shell", "--pure")
		cmd.Args = append(cmd.Args, toKeepArgs(env, buildAllowList(s.env))...)
		cmd.Args = append(cmd.Args, nixShellFilePath)
		cmd.Env = env
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		debug.Log("Unable to detect the user's shell, falling back to: %v", cmd.Args)
		return errors.WithStack(cmd.Run())
	}

	var cmd *exec.Cmd
	if featureflag.UnifiedEnv.Enabled() {
		shellrc, err := s.writeDevboxShellrc()
		if err != nil {
			// We don't have a good fallback here, since all the variables we need for anything to work
			// are in the shellrc file. For now let's fail. Later on, we should remove the vars from the
			// shellrc file. That said, one of the variables we have to evaluate ($shellHook), so we need
			// the shellrc file anyway (unless we remove the hook somehow).
			debug.Log("Failed to write devbox shellrc: %s", err)
			return errors.WithStack(err)
		}
		// Link other files that affect the shell settings and environments.
		s.linkShellStartupFiles(filepath.Dir(shellrc))
		extraEnv, extraArgs := s.shellRCOverrides(shellrc)

		cmd = exec.Command(s.binPath)
		cmd.Env = append(s.env, extraEnv...)
		cmd.Args = append(cmd.Args, extraArgs...)
		debug.Log("Executing shell %s with args: %v", s.binPath, cmd.Args)
	} else {
		// Use nix-shell
		cmd = exec.Command("nix-shell", "--command", s.execCommand(), "--pure")
		keepArgs := toKeepArgs(env, buildAllowList(s.env))
		cmd.Args = append(cmd.Args, keepArgs...)
		cmd.Args = append(cmd.Args, nixShellFilePath)
		cmd.Env = env
		debug.Log("Executing nix-shell command: %v", cmd.Args)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	// If the error is an ExitError, this means the shell started up fine but there was
	// an error from executing a shell command or script.
	//
	// This could be from one of the generated shellrc commands, but more likely is from
	// a user's command or script. So, we want to return nil for this.
	if exitErr := (&exec.ExitError{}); errors.As(err, &exitErr) {

		// The exception to the previous comment is if we are executing a shell script
		// via `devbox run` or the deprecated `devbox shell -- <command>`. In this case,
		// we do want to return the exit code of the script that was run.
		if s.ScriptCommand != "" {
			return usererr.NewExecError(err)
		}
		return nil
	}

	// This means that there was a error from devbox's code or nix's code. Not a user
	// error and so we do return it.
	return errors.WithStack(err)
}

// execCommand is a command that replaces the current shell with s. This is what
// Run sets the nix-shell --command flag to.
func (s *Shell) execCommand() string {
	// We exec env, which will then exec the shell. This lets us set
	// additional environment variables before any of the shell's init
	// scripts run.
	args := []string{

		"exec",
		"env",

		// Correct SHELL to be the one we're about to exec.
		fmt.Sprintf(`"SHELL=%s"`, s.binPath),
	}

	// userShellrcPath is empty when we know the path to the user's shell,
	// but we don't recognize its name. In this case we don't know how to
	// override the shellrc file, so just launch the shell without any
	// additional args.
	if s.userShellrcPath == "" {
		args = append(args, s.binPath)
		if s.ScriptCommand != "" {
			args = append(args, "-ic", shellescape.Quote(s.ScriptCommand))
		}
		return strings.Join(args, " ")
	}

	// Create a devbox shellrc file that runs the user's shellrc + the shell
	// hook in devbox.json.
	shellrc, err := s.writeDevboxShellrc()
	if err != nil {
		// Fall back to just launching the shell without a custom
		// shellrc.
		debug.Log("Failed to write devbox shellrc: %v", err)
		return strings.Join(append(args, s.binPath), " ")
	}

	// Link other files that affect the shell settings and environments.
	s.linkShellStartupFiles(filepath.Dir(shellrc))

	extraEnv, extraArgs := s.shellRCOverrides(shellrc)
	args = append(args, extraEnv...)
	args = append(args, s.binPath)
	args = append(args, extraArgs...)
	if s.ScriptCommand != "" {
		args = append(args, "-ic")
		args = append(args, "run_script")
	}
	return strings.Join(args, " ")
}

func (s *Shell) RunInShell() error {
	env := append(
		os.Environ(),
		// Prevent the user's shellrc from re-sourcing nix-daemon.sh
		// inside the devbox shell.
		"__ETC_PROFILE_NIX_SOURCED=1",
		"NIXPKGS_ALLOW_UNFREE=1",
	)
	debug.Log("Running inside devbox shell with environment: %v", env)
	cmd := exec.Command(s.execCommandInShell())
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	debug.Log("Executing command from inside devbox shell: %v", cmd.Args)

	return errors.WithStack(usererr.NewExecError(cmd.Run()))
}

func (s *Shell) shellRCOverrides(shellrc string) (extraEnv []string, extraArgs []string) {
	// Shells have different ways of overriding the shellrc, so we need to
	// look at the name to know which env vars or args to set when launching the shell.
	switch s.name {
	case shBash:
		extraArgs = []string{"--rcfile", shellescape.Quote(shellrc)}
	case shZsh:
		extraEnv = []string{fmt.Sprintf(`ZDOTDIR=%s`, shellescape.Quote(filepath.Dir(shellrc)))}
	case shKsh, shPosix:
		extraEnv = []string{fmt.Sprintf(`ENV=%s`, shellescape.Quote(shellrc))}
	}
	return extraEnv, extraArgs
}

func (s *Shell) execCommandInShell() (string, string, string) {
	args := []string{}

	if s.ScriptCommand != "" {
		args = append(args, "-ic")
	}
	return s.binPath, strings.Join(args, " "), s.ScriptCommand
}

func (s *Shell) writeDevboxShellrc() (path string, err error) {

	// We need a temp dir (as opposed to a temp file) because zsh uses
	// ZDOTDIR to point to a new directory containing the .zshrc.
	tmp, err := os.MkdirTemp("", "devbox")
	if err != nil {
		return "", fmt.Errorf("create temp dir for shell init file: %v", err)
	}

	// This is a best-effort to include the user's existing shellrc.
	userShellrc := []byte{}
	if s.userShellrcPath != "" {
		// If we can't read it, then just omit it from the devbox shellrc.
		userShellrc, _ = os.ReadFile(s.userShellrcPath)
	}

	// If the user already has a shellrc file, then give the devbox shellrc
	// file the same name. Otherwise, use an arbitrary name of "shellrc".
	shellrcName := "shellrc"
	if s.userShellrcPath != "" {
		shellrcName = filepath.Base(s.userShellrcPath)
	}
	path = filepath.Join(tmp, shellrcName)
	shellrcf, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("write to shell init file: %v", err)
	}
	defer func() {
		cerr := shellrcf.Close()
		if err == nil {
			err = cerr
		}
	}()

	pathPrepend := s.profileDir + "/bin"
	if s.pkgConfigDir != "" {
		pathPrepend = s.pkgConfigDir + ":" + pathPrepend
	}

	err = shellrcTmpl.Execute(shellrcf, struct {
		ProjectDir       string
		OriginalInit     string
		OriginalInitPath string
		UserHook         string
		PluginInitHook   string
		PathPrepend      string
		ScriptCommand    string
		ShellStartTime   string
		HistoryFile      string
		UnifiedEnv       bool
	}{
		ProjectDir:       s.projectDir,
		OriginalInit:     string(bytes.TrimSpace(userShellrc)),
		OriginalInitPath: filepath.Clean(s.userShellrcPath),
		UserHook:         strings.TrimSpace(s.UserInitHook),
		PluginInitHook:   strings.TrimSpace(s.pluginInitHook),
		PathPrepend:      pathPrepend,
		ScriptCommand:    strings.TrimSpace(s.ScriptCommand),
		ShellStartTime:   s.shellStartTime,
		HistoryFile:      strings.TrimSpace(s.historyFile),
		UnifiedEnv:       featureflag.UnifiedEnv.Enabled(),
	})
	if err != nil {
		return "", fmt.Errorf("execute shellrc template: %v", err)
	}

	debug.Log("Wrote devbox shellrc to: %s", path)
	return path, nil
}

// linkShellStartupFiles will link files used by the shell for initialization.
// We choose to link instead of copy so that changes made outside can be reflected
// within the devbox shell.
//
// We do not link the .{shell}rc files, since devbox modifies them. See writeDevboxShellrc
func (s *Shell) linkShellStartupFiles(shellSettingsDir string) {

	// For now, we only need to do this for zsh shell
	if s.name == shZsh {
		// Useful explanation of zsh startup files: https://zsh.sourceforge.io/FAQ/zshfaq03.html#l20
		filenames := []string{".zshenv", ".zprofile", ".zlogin"}
		for _, filename := range filenames {
			fileOld := filepath.Join(filepath.Dir(s.userShellrcPath), filename)
			if _, err := os.Stat(fileOld); errors.Is(err, fs.ErrNotExist) {
				// this file may not be relevant for the user's setup.
				continue
			} else if err != nil {
				debug.Log("os.Stat error for %s is %v", fileOld, err)
			}

			fileNew := filepath.Join(shellSettingsDir, filename)

			if err := os.Link(fileOld, fileNew); err == nil {
				debug.Log("Linked shell startup file %s to %s", fileOld, fileNew)
			} else {
				// This is a best-effort operation. If there's an error then log it for visibility but continue.
				debug.Log("Error linking zsh setting file from %s to %s: %v", fileOld, fileNew, err)
			}
		}
	}
}

// envToKeep is the set of environment variables that we want to copy verbatim
// to the new devbox shell.
var envToKeep = map[string]bool{
	// POSIX
	//
	// Variables that are part of the POSIX standard.
	"HOME":   true,
	"OLDPWD": true,
	"PWD":    true,
	"TERM":   true,
	"TZ":     true,
	"USER":   true,

	// POSIX Locale
	//
	// Variables that are part of the POSIX standard which define
	// the shell's locale.
	"LC_ALL":      true, // Sets and overrides all of the variables below.
	"LANG":        true, // Default to use for any of the variables below that are unset or null.
	"LC_COLLATE":  true, // Collation order.
	"LC_CTYPE":    true, // Character classification and case conversion.
	"LC_MESSAGES": true, // Formats of informative and diagnostic messages and interactive responses.
	"LC_MONETARY": true, // Monetary formatting.
	"LC_NUMERIC":  true, // Numeric, non-monetary formatting.
	"LC_TIME":     true, // Date and time formats.

	// Common
	//
	// Variables that most programs agree on, but aren't strictly
	// part of POSIX.
	"TERM_PROGRAM":         true, // Name of the terminal the shell is running in.
	"TERM_PROGRAM_VERSION": true, // The version of TERM_PROGRAM.
	"SHLVL":                true, // The number of nested shells.

	// Apple Terminal
	//
	// Special-cased variables that macOS's Terminal.app sets before
	// launching the shell. It's not clear what exactly all of these do,
	// but it seems like omitting them can cause problems.
	"TERM_SESSION_ID":        true,
	"SHELL_SESSIONS_DISABLE": true, // Respect session save/resume setting (see /etc/zshrc_Apple_Terminal).
	"SECURITYSESSIONID":      true,

	// SSH variables
	"SSH_TTY": true, // Used by devbox telemetry logging

	// Nix + Devbox
	//
	// Variables specific to running in a Nix shell and devbox shell.
	"PARENT_PATH":               true, // The PATH of the parent shell (where `devbox shell` was invoked).
	"__ETC_PROFILE_NIX_SOURCED": true, // Prevents Nix from being sourced again inside a devbox shell.
	"NIX_SSL_CERT_FILE":         true, // The path to Nix-installed SSL certificates (used by some Nix programs).
	"SSL_CERT_FILE":             true, // The path to non-Nix SSL certificates (used by some Nix and non-Nix programs).
	"NIXPKGS_ALLOW_UNFREE":      true, // Whether to allow the use of unfree packages.

	// Devbox
	//
	// Variables specific to devbox configuration.
	"DEVBOX_USE_VERSION": true, // Version of devbox used upon invoking `devbox shell`.
	"DEVBOX_REGION":      true, // Region of the Devbox cloud
}

func buildAllowList(allowList []string) map[string]bool {
	for _, kv := range allowList {
		key, _, _ := strings.Cut(kv, "=")
		envToKeep[key] = true
	}
	return envToKeep
}

func filterVars(env []string, allowList map[string]bool) []string {
	vars := make([]string, 0, len(allowList))
	for _, kv := range env {
		key, _, _ := strings.Cut(kv, "=")
		if allowList[key] {
			vars = append(vars, kv)
		}
	}
	return vars
}

// toKeepArgs takes a slice of environment variables in key=value format and
// builds a slice of "--keep" arguments that tell nix-shell which ones to
// keep.
//
// See envToKeep for the full set of permanent kept environment variables.
// We also --keep any variables set by package configuration.
func toKeepArgs(env []string, allowList map[string]bool) []string {
	args := make([]string, 0, len(allowList)*2)
	for _, kv := range filterVars(env, allowList) {
		key, _, _ := strings.Cut(kv, "=")
		args = append(args, "--keep", key)
	}
	return args
}

// splitNixList splits and cleans a list of space-delimited paths. It is similar
// to filepath.SplitList for Nix environment variables, which do not use
// filepath.ListSeparator.
func splitNixList(s string) []string {
	split := strings.Fields(s)
	for i, dir := range split {
		split[i] = filepath.Clean(dir)
	}
	return split
}

// CleanEnvPath takes a string formatted as a shell PATH and cleans it for
// passing to nix-shell. It does the following rules for each entry:
//
//  1. Applies filepath.Clean.
//  2. Removes the path if it's relative (must begin with '/' and not be '.').
//  3. Removes the path if it's a descendant of a user Nix profile directory
//     (the default Nix profile is kept).
func CleanEnvPath(pathEnv string, nixProfilesEnv string) string {
	// Just to be safe, we need to guarantee that the NIX_PROFILES paths
	// have been filepath.Clean'ed. The shellrc.tmpl has some commands that
	// assume they are.
	nixProfileDirs := splitNixList(nixProfilesEnv)

	split := filepath.SplitList(pathEnv)
	if len(split) == 0 {
		return ""
	}

	cleaned := make([]string, 0, len(split))
	for _, path := range split {
		path = filepath.Clean(path)
		if path == "." || path[0] != '/' {
			// Don't allow relative paths.
			continue
		}

		keep := true
		for _, profileDir := range nixProfileDirs {
			// nixProfileDirs may be of the form: /nix/var/nix/profile/default or
			// $HOME/.nix-profile. The former contains Nix itself (nix-store, nix-env,
			// etc.), which we want to keep so it's available in the shell. The latter
			// contains programs that the user installed with Nix, which we want to filter
			// out so that only devbox-managed Nix packages are available.
			isProfileDir := strings.HasPrefix(path, profileDir)
			isSystemProfile := strings.HasPrefix(profileDir, "/nix")
			if isProfileDir && !isSystemProfile {
				keep = false
				break
			}
		}

		if keep {
			cleaned = append(cleaned, path)
		}
	}

	return strings.Join(cleaned, string(filepath.ListSeparator))
}
