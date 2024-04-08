// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package openssh

import (
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"text/template"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/fileutil"
)

// These must match what's in sshConfigTmpl. We should eventually make the hosts
// a template variable.
const (
	gatewayProdHost = "gateway.devbox.sh"
	gatewayDevHost  = "gateway.dev.devbox.sh"
)

//go:embed sshconfig.tmpl
var sshConfigText string
var sshConfigTmpl = template.Must(template.New("sshconfig").Parse(sshConfigText))

//go:embed known_hosts
var sshKnownHosts []byte

// SetupDevbox updates the user's OpenSSH configuration so that they can connect
// to Devbox Cloud hosts. It does nothing if Devbox Cloud is already
// configured.
func SetupDevbox() error {
	return setupDevbox("", 0)
}

// SetupInsecureDebug is like SetupDevbox, but also configures an additional
// gateway with host key checking disabled. If gatewayAddr is a
// well-known *.devbox.sh gateway, then SetupInsecureDebug doesn't add any
// extra hosts and acts identically to SetupDevbox.
func SetupInsecureDebug(gatewayAddr string) error {
	host, port := splitHostPort(gatewayAddr)
	if host != gatewayProdHost && host != gatewayDevHost {
		return setupDevbox(host, port)
	}
	return setupDevbox("", 0)
}

func setupDevbox(debugHost string, debugPort int) error {
	devboxSSHDir, err := devboxSSHDir()
	if err != nil {
		return err
	}

	// Ensure ~/.config/devbox/ssh/sockets exists.
	if _, err := devboxSocketsDir(); err != nil {
		return err
	}

	// Try to remove any old debug host keys. It's okay if this fails.
	devboxKnownHostsDebug := filepath.Join(devboxSSHDir, "known_hosts_debug")
	_ = os.Remove(devboxKnownHostsDebug)

	devboxKnownHostsPath := filepath.Join(devboxSSHDir, "known_hosts")
	devboxKnownHosts, err := editFile(devboxKnownHostsPath, 0o644)
	if err != nil {
		return err
	}
	defer devboxKnownHosts.Close()
	if _, err := devboxKnownHosts.Write(sshKnownHosts); err != nil {
		return err
	}
	if err := devboxKnownHosts.Commit(); err != nil {
		return err
	}

	devboxIncludePath := filepath.Join(devboxSSHDir, "config")
	devboxSSHConfig, err := editFile(devboxIncludePath, 0o644)
	if err != nil {
		return err
	}
	defer devboxSSHConfig.Close()

	tmplData := struct {
		ConfigVersion string
		ConfigDir     string
		DebugGateway  struct {
			Host string
			Port int
		}
	}{
		ConfigVersion: "0.0.1",
		ConfigDir:     devboxSSHDir,
	}
	tmplData.DebugGateway.Host = debugHost
	tmplData.DebugGateway.Port = debugPort
	err = errors.WithStack(sshConfigTmpl.Execute(devboxSSHConfig, tmplData))
	if err != nil {
		return errors.WithStack(err)
	}
	if err := devboxSSHConfig.Commit(); err != nil {
		return err
	}
	if err := updateUserSSHConfig(devboxIncludePath); err != nil {
		return err
	}

	// Create the known_hosts_debug file with the correct permissions if a
	// debug gateway is configured. It's okay if this fails because it's
	// only used for debugging.
	if debugHost != "" {
		f, err := os.OpenFile(devboxKnownHostsDebug, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err == nil {
			f.Close()
		}
	}
	return nil
}

// AddVMKey sets the private SSH key for the given Devbox VM host. If a key was
// previously set for the host, AddVMKey replaces it with the new key. The old
// key is not recoverable.
//
// AddVMKey only manages keys specific to Devbox Cloud. It will not touch any of
// the user's keys in ~/.ssh.
func AddVMKey(hostname, key string) error {
	keysDir, err := devboxKeysDir()
	if err != nil {
		return err
	}
	keyFile, err := editFile(filepath.Join(keysDir, hostname), 0o600)
	if err != nil {
		return err
	}
	defer keyFile.Close()

	if _, err := io.WriteString(keyFile, key); err != nil {
		return err
	}
	return keyFile.Commit()
}

func updateUserSSHConfig(devboxIncludePath string) (err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return errors.WithStack(err)
	}
	dotSSH := filepath.Join(home, ".ssh")
	if err := EnsureDirExists(dotSSH, 0o700, true); err != nil {
		return err
	}

	sshConfig, err := editFile(filepath.Join(dotSSH, "config"), 0o644)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := sshConfig.Close()
		if err == nil {
			err = closeErr
		}
	}()

	bufw := bufio.NewWriter(sshConfig)
	_, err = fmt.Fprintf(bufw, "Include \"%s\"\n", devboxIncludePath)
	if err != nil {
		return err
	}

	// Look for an existing Include directive, copying the file contents as
	// we read.
	if containsDevboxInclude(io.TeeReader(sshConfig, bufw)) {
		// We found an existing Include - don't save and return.
		return nil
	}
	// We didn't find an existing Include - copy the rest of the user's SSH
	// config and then commit the changes.
	if _, err := bufw.ReadFrom(sshConfig); err != nil {
		return errors.WithStack(err)
	}
	if err := bufw.Flush(); err != nil {
		return errors.WithStack(err)
	}
	return sshConfig.Commit()
}

var (
	reDevboxInclude = regexp.MustCompile(`(?i)^[ \t]*"?Include"?[ \t=][^#]*devbox/ssh/config`)
	reHostOrMatch   = regexp.MustCompile(`(?i)[ \t]*"?(Host|Match) `)
)

func containsDevboxInclude(r io.Reader) bool {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Bytes()
		if reDevboxInclude.Match(line) {
			return true
		}

		// Unconditional Include directives must come before any Host or
		// Match blocks. If we found one of those blocks then we've gone
		// too far.
		if reHostOrMatch.Match(line) {
			return false
		}
	}
	return false
}

func EnsureDirExists(path string, perm fs.FileMode, chmod bool) error {
	return fileutil.EnsureDirExists(path, perm, chmod)
}

// returns path to ~/.config/devbox/ssh
func devboxSSHDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.WithStack(err)
	}

	// Ensure ~/.config exists but don't touch existing permissions.
	dotConfig := filepath.Join(home, ".config")
	if err := EnsureDirExists(dotConfig, 0o755, false); err != nil {
		return "", err
	}

	// Ensure ~/.config/devbox exists and force permissions to 0755.
	devboxConfigDir := filepath.Join(dotConfig, "devbox")
	if err := EnsureDirExists(devboxConfigDir, 0o755, true); err != nil {
		return "", err
	}

	// Ensure ~/.config/devbox/ssh exists and force permissions to 0700.
	devboxSSHDir := filepath.Join(devboxConfigDir, "ssh")
	if err := EnsureDirExists(devboxSSHDir, 0o700, true); err != nil {
		return "", err
	}
	return devboxSSHDir, nil
}

func devboxKeysDir() (string, error) {
	sshDir, err := devboxSSHDir()
	if err != nil {
		return "", err
	}
	keysDir := filepath.Join(sshDir, "keys")
	if err := EnsureDirExists(keysDir, 0o700, true); err != nil {
		return "", err
	}
	return keysDir, nil
}

func devboxSocketsDir() (string, error) {
	sshDir, err := devboxSSHDir()
	if err != nil {
		return "", err
	}
	sockets := filepath.Join(sshDir, "sockets")
	if err := EnsureDirExists(sockets, 0o700, true); err != nil {
		return "", err
	}
	return sockets, nil
}

// atomicEdit reads from a source file and writes changes to a separate
// temporary file. Upon a call to Commit, it atomically overwrites the source
// file with the temp file, guaranteeing that all of the file Writes succeed or
// none at all. Calling Close before calling Commit discards any written data,
// leaving the source file untouched.
type atomicEdit struct {
	path     string
	editFile *os.File
	tmpFile  *os.File

	closed bool
	err    error
}

// editFile opens the file at path for editing. Writes to atomicEdit will not
// modify the file until Commit is called. If the file doesn't exist, calls to
// Read immediately return io.EOF and Commit will create it with permissions
// perm. If the file does exist, Commit atomically applies any written data and
// changes its permissions to perm.
//
// Calling Close without calling Commit discards all written data. It is
// unnecessary but valid to call Close after Commit. This makes it easier to
// defer closing the file.
func editFile(path string, perm os.FileMode) (*atomicEdit, error) {
	// editFile will be nil when creating a new file.
	editFile, err := os.Open(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, errors.WithStack(err)
	}

	// Atomic file renames require that both files are on the same volume.
	// Putting the tmp file in the same directory is the best way to ensure
	// that happens.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".devbox")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Make sure to set permissions before writing anything. This also means
	// perm must be user-writeable.
	if err := tmp.Chmod(perm); err != nil {
		return nil, errors.WithStack(err)
	}
	return &atomicEdit{
		path:     path,
		editFile: editFile,
		tmpFile:  tmp,
	}, nil
}

func (a *atomicEdit) Read(p []byte) (n int, err error) {
	if a.editFile == nil {
		return 0, io.EOF
	}
	n, err = a.editFile.Read(p)

	// Don't use `errors.Is` here because we only want to avoid wrapping
	// io.EOF directly. This is for compatibility with the io.Writer
	// interface.
	// nolint:errorlint
	if err != nil && err != io.EOF {
		err = errors.WithStack(err)
	}
	return n, err
}

func (a *atomicEdit) Write(p []byte) (n int, err error) {
	n, err = a.tmpFile.Write(p)

	// Don't use `errors.Is` here because we only want to avoid wrapping
	// io.EOF directly. This is for compatibility with the io.Writer
	// interface.
	// nolint:errorlint
	if err != nil && err != io.EOF {
		err = errors.WithStack(err)
	}
	return n, err
}

func (a *atomicEdit) Commit() error {
	if a.closed {
		return a.err
	}
	a.closed = true

	if a.editFile != nil {
		// Ignore close errors - we only ever read from the original
		// file.
		a.editFile.Close()
	}
	if a.err = errors.WithStack(a.tmpFile.Close()); a.err != nil {
		return a.err
	}
	if a.err = errors.WithStack(os.Rename(a.tmpFile.Name(), a.path)); a.err != nil {
		return a.err
	}
	return a.err
}

func (a *atomicEdit) Close() error {
	if a.closed {
		return a.err
	}
	a.closed = true

	// Ignore close errors - we're throwing away any changes.
	if a.editFile != nil {
		a.editFile.Close()
	}
	a.tmpFile.Close()
	a.err = errors.WithStack(os.Remove(a.tmpFile.Name()))
	return a.err
}
