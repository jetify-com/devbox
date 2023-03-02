// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"bytes"
	_ "embed"
	"encoding/json"
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
	"go.jetpack.io/devbox/internal/planner/plansdk"
	"go.jetpack.io/devbox/internal/xdg"
)

//go:embed shellrc.tmpl
var shellrcText string
var shellrcTmpl = template.Must(template.New("shellrc").Parse(shellrcText))

//go:embed shellrc_fish.tmpl
var fishrcText string
var fishrcTmpl = template.Must(template.New("shellrc_fish").Parse(fishrcText))

type name string

const (
	shUnknown name = ""
	shBash    name = "bash"
	shZsh     name = "zsh"
	shKsh     name = "ksh"
	shFish    name = "fish"
	shPosix   name = "posix"
)

var ErrNoRecognizableShellFound = errors.New(
	"SHELL in undefined, and couldn't find any common shells in PATH")

// TODO move to `impl` package. This is no longer a pure nix shell.
// Also consider splitting this struct's functionality so that there is a simpler
// `nix.Shell` that can produce a raw nix shell once again.
//
// DevboxShell configures a user's shell to run in Devbox. Its zero value is a
// fallback shell that launches a regular Nix shell.
type DevboxShell struct {
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

type ShellOption func(*DevboxShell)

// NewDevboxShell initializes the DevboxShell struct so it can be used to start a shell environment
// for the devbox project.
func NewDevboxShell(nixpkgsCommitHash string, opts ...ShellOption) (*DevboxShell, error) {
	shPath, err := shellPath(nixpkgsCommitHash)
	if err != nil {
		return nil, err
	}
	sh := initShellBinaryFields(shPath)

	for _, opt := range opts {
		opt(sh)
	}

	debug.Log("Recognized shell as: %s", sh.binPath)
	debug.Log("Looking for user's shell init file at: %s", sh.userShellrcPath)
	return sh, nil
}

// shellPath returns the path to a shell binary, or error if none found.
func shellPath(nixpkgsCommitHash string) (path string, err error) {
	defer func() {
		if err != nil {
			path = filepath.Clean(path)
		}
	}()

	// First, check the SHELL environment variable.
	path = os.Getenv("SHELL")
	if path != "" {
		debug.Log("Using SHELL env var for shell binary path: %s\n", path)
		return path, nil
	}

	// Second, fallback to using the bash that nix uses by default.

	var bashNixStorePath string // of the form /nix/store/{hash}-bash-{version}/
	if featureflag.Flakes.Enabled() {
		cmd := exec.Command(
			"nix", "eval", "--raw",
			fmt.Sprintf("%s#bash", FlakeNixpkgs(nixpkgsCommitHash)),
		)
		cmd.Args = append(cmd.Args, ExperimentalFlags()...)
		out, err := cmd.Output()
		if err != nil {
			return "", errors.WithStack(err)
		}
		bashNixStorePath = string(out)
	} else {
		nixpkgsInfo, err := plansdk.GetNixpkgsInfo(nixpkgsCommitHash)
		if err != nil {
			return "", err
		}
		expr := fmt.Sprintf(
			"let pkgs = import (fetchTarball { url = \"%s\"; }) {}; in {inherit(pkgs.bash) outPath; }",
			nixpkgsInfo.URL,
		)
		cmd := exec.Command("nix-instantiate", "--eval", "--strict",
			"--json",
			"--expr", expr,
		)
		out, err := cmd.Output()
		if err != nil {
			return "", errors.WithStack(err)
		}
		if err := json.Unmarshal(out, &bashNixStorePath); err != nil {
			return "", errors.WithStack(err)
		}
	}

	if bashNixStorePath != "" {
		// the output is the raw path to the bash installation in the /nix/store
		return fmt.Sprintf("%s/bin/bash", bashNixStorePath), nil
	}

	// Else, return an error
	return "", ErrNoRecognizableShellFound
}

// initShellBinaryFields initializes the fields specific to the shell binary that will be used
// for the devbox shell.
func initShellBinaryFields(path string) *DevboxShell {

	shell := &DevboxShell{binPath: path}
	base := filepath.Base(path)
	// Login shell
	if base[0] == '-' {
		base = base[1:]
	}
	switch base {
	case "bash":
		shell.name = shBash
		shell.userShellrcPath = rcfilePath(".bashrc")
	case "zsh":
		shell.name = shZsh
		shell.userShellrcPath = rcfilePath(".zshrc")
	case "ksh":
		shell.name = shKsh
		shell.userShellrcPath = rcfilePath(".kshrc")
	case "fish":
		shell.name = shFish
		shell.userShellrcPath = fishConfig()
	case "dash", "ash", "shell":
		shell.name = shPosix
		shell.userShellrcPath = os.Getenv("ENV")

		// Just make up a name if there isn't already an init file set
		// so we have somewhere to put a new one.
		if shell.userShellrcPath == "" {
			shell.userShellrcPath = ".shinit"
		}
	default:
		shell.name = shUnknown
	}
	return shell
}

// If/once we end up making plugins the same as devbox.json we probably want
// to merge all init hooks into single field
func WithPluginInitHook(hook string) ShellOption {
	return func(s *DevboxShell) {
		s.pluginInitHook = hook
	}
}

func WithProfile(profileDir string) ShellOption {
	return func(s *DevboxShell) {
		s.profileDir = profileDir
	}
}

func WithHistoryFile(historyFile string) ShellOption {
	return func(s *DevboxShell) {
		s.historyFile = historyFile
	}
}

// TODO: Consider removing this once plugins add env vars directly to binaries
// via wrapper scripts.
func WithEnvVariables(envVariables map[string]string) ShellOption {
	return func(s *DevboxShell) {
		for k, v := range envVariables {
			s.env = append(s.env, fmt.Sprintf("%s=%s", k, v))
		}
	}
}

func WithUserScript(name string, command string) ShellOption {
	return func(s *DevboxShell) {
		s.ScriptName = name
		s.ScriptCommand = command
	}
}

func WithPKGConfigDir(pkgConfigDir string) ShellOption {
	return func(s *DevboxShell) {
		s.pkgConfigDir = pkgConfigDir
	}
}

func WithProjectDir(projectDir string) ShellOption {
	return func(s *DevboxShell) {
		s.projectDir = projectDir
	}
}

func WithShellStartTime(time string) ShellOption {
	return func(s *DevboxShell) {
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

func fishConfig() string {
	return filepath.Join(xdg.ConfigDir(), "fish", "config.fish")
}

func (s *DevboxShell) Run(nixShellFilePath, nixFlakesFilePath string) error {
	// Copy the current PATH into nix-shell, but clean and remove some
	// directories that are incompatible.
	parentPath := JoinPathLists(os.Getenv("PATH"))

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
func (s *DevboxShell) execCommand() string {
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

func (s *DevboxShell) RunInShell() error {
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

func (s *DevboxShell) shellRCOverrides(shellrc string) (extraEnv []string, extraArgs []string) {
	// Shells have different ways of overriding the shellrc, so we need to
	// look at the name to know which env vars or args to set when launching the shell.
	switch s.name {
	case shBash:
		extraArgs = []string{"--rcfile", shellescape.Quote(shellrc)}
	case shZsh:
		extraEnv = []string{fmt.Sprintf(`ZDOTDIR=%s`, shellescape.Quote(filepath.Dir(shellrc)))}
	case shKsh, shPosix:
		extraEnv = []string{fmt.Sprintf(`ENV=%s`, shellescape.Quote(shellrc))}
	case shFish:
		if featureflag.UnifiedEnv.Enabled() {
			extraArgs = []string{"-C", ". " + shellrc}
		} else {
			// Needs quotes because it's wrapped inside the nix-shell command
			extraArgs = []string{"-C", shellescape.Quote(". " + shellrc)}
		}
	}
	return extraEnv, extraArgs
}

func (s *DevboxShell) execCommandInShell() (string, string, string) {
	args := []string{}

	if s.ScriptCommand != "" {
		args = append(args, "-ic")
	}
	return s.binPath, strings.Join(args, " "), s.ScriptCommand
}

func (s *DevboxShell) writeDevboxShellrc() (path string, err error) {

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

	tmpl := shellrcTmpl
	if s.name == shFish {
		tmpl = fishrcTmpl
	}

	exportEnv := ""
	if featureflag.UnifiedEnv.Enabled() {
		strb := strings.Builder{}
		for _, kv := range s.env {
			k, v, ok := strings.Cut(kv, "=")
			if !ok {
				continue
			}
			strb.WriteString("export ")
			strb.WriteString(k)
			strb.WriteString(`="`)
			for _, r := range v {
				switch r {
				// Special characters inside double quotes:
				// https://pubs.opengroup.org/onlinepubs/009604499/utilities/xcu_chap02.html#tag_02_02_03
				case '$', '`', '"', '\\', '\n':
					strb.WriteRune('\\')
				}
				strb.WriteRune(r)
			}
			strb.WriteString("\"\n")
		}
		exportEnv = strings.TrimSpace(strb.String())
	}

	err = tmpl.Execute(shellrcf, struct {
		ProjectDir       string
		OriginalInit     string
		OriginalInitPath string
		UserHook         string
		PluginInitHook   string
		PathPrepend      string
		ScriptCommand    string
		ShellStartTime   string
		HistoryFile      string
		ExportEnv        string
	}{
		ProjectDir:       s.projectDir,
		OriginalInit:     string(bytes.TrimSpace(userShellrc)),
		OriginalInitPath: s.userShellrcPath,
		UserHook:         strings.TrimSpace(s.UserInitHook),
		PluginInitHook:   strings.TrimSpace(s.pluginInitHook),
		PathPrepend:      pathPrepend,
		ScriptCommand:    strings.TrimSpace(s.ScriptCommand),
		ShellStartTime:   s.shellStartTime,
		HistoryFile:      strings.TrimSpace(s.historyFile),
		ExportEnv:        exportEnv,
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
func (s *DevboxShell) linkShellStartupFiles(shellSettingsDir string) {

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

// JoinPathLists joins and cleans PATH-style strings of
// [os.ListSeparator] delimited paths. To clean a path list, it splits it and
// does the following for each element:
//
//  1. Applies [filepath.Clean].
//  2. Removes the path if it's relative (must begin with '/' and not be '.').
//  3. Removes the path if it's a duplicate.
func JoinPathLists(pathLists ...string) string {
	if len(pathLists) == 0 {
		return ""
	}

	seen := make(map[string]bool)
	var cleaned []string
	for _, path := range pathLists {
		for _, path := range filepath.SplitList(path) {
			path = filepath.Clean(path)
			if path == "." || path[0] != '/' {
				// Remove empty paths and don't allow relative
				// paths for security reasons.
				continue
			}
			if !seen[path] {
				cleaned = append(cleaned, path)
			}
			seen[path] = true
		}
	}
	return strings.Join(cleaned, string(filepath.ListSeparator))
}
