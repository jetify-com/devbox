// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package sshconfig

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/pkg/errors"
)

//go:embed sshconfig.tmpl
var sshConfigText string
var sshConfigTmpl = template.Must(template.New("sshconfig").Parse(sshConfigText))

func Setup() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return errors.WithStack(err)
	}

	dir, err := ensureDirectoryExists(home)
	if err != nil {
		return err
	}

	configFilePath, err := writeConfigFile(dir)
	if err != nil {
		return err
	}

	err = writeIncludeInGlobalConfig(home, configFilePath)
	if err != nil {
		return err
	}

	return nil
}

func ensureDirectoryExists(home string) (string, error) {
	configDir := filepath.Join(home, ".config")

	dir := filepath.Join(configDir, "devbox/ssh")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", errors.WithStack(err)
	}

	// MkdirAll is a no-op for pre-existing directories, so lets ensure the permissions are correct.
	if err := os.Chmod(dir, 0700); err != nil {
		return "", errors.WithStack(err)
	}

	return dir, nil
}

func writeConfigFile(dir string) (string, error) {

	filePath := filepath.Join(dir, "config")
	sshConfigFile, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return "", errors.WithStack(err)
	}
	defer func() {
		// deliberately ignore the error here.
		_ = sshConfigFile.Close()
	}()

	err = sshConfigTmpl.Execute(sshConfigFile, struct {
		ConfigVersion string
		ConfigDir     string
	}{
		ConfigVersion: "0.0.1",
		ConfigDir:     dir,
	})
	return filePath, errors.WithStack(err)
}

func writeIncludeInGlobalConfig(home string, devboxSSHConfigFilePath string) error {

	configFilePath := filepath.Join(home, ".ssh/config")

	// Ensure the ~/.ssh/config file exists
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		if err = os.MkdirAll(filepath.Join(home, ".ssh"), 0700); err != nil {
			return errors.WithStack(err)
		}
	}

	// Read the ssh config contents
	configFile, err := os.OpenFile(configFilePath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		// deliberately ignore the error here
		_ = configFile.Close()
	}()

	buf := &bytes.Buffer{}
	_, err = buf.ReadFrom(configFile)
	if err != nil {
		return errors.WithStack(err)
	}
	configContents := buf.String()

	// if the Include directive is present, then our work is done
	if containsDevboxIncludeDirective(configContents) {
		return nil
	}

	// Set the Include directive
	configContents = fmt.Sprintf("Include %s\n\n%s", devboxSSHConfigFilePath, configContents)
	err = os.WriteFile(configFilePath, []byte(configContents), 0644)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

var reDevboxIncludeDirective = regexp.MustCompile("Include.*devbox/ssh")

func containsDevboxIncludeDirective(configContents string) bool {
	for _, line := range strings.Split(configContents, "\n") {
		if reDevboxIncludeDirective.MatchString(line) {
			return true
		}
	}
	return false
}
