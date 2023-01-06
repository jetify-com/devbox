package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/fileutil"
)

type UsernameSource string

const (
	// Logged out:
	NoneSource   UsernameSource = "none"
	GithubSource UsernameSource = "github"
)

type Info struct {
	// Source of the username
	Source string `json:"source,omitempty"`

	Username string `json:"username,omitempty"`

	// Time we set this value (unixtime)
	Time string `json:"time,omitempty"`
}

func Username() (UsernameSource, string, error) {
	isLoggedIn, err := authFileExists()
	if err != nil {
		return "", "", err
	}
	if !isLoggedIn {
		return NoneSource, "", nil
	}
	info, err := read()
	if err != nil {
		return "", "", errors.WithStack(err)
	}
	return UsernameSource(info.Source), info.Username, nil
}

func SaveUsername(source UsernameSource, username string) error {

	isLoggedIn, err := authFileExists()
	if err != nil {
		return err
	}

	info := &Info{}
	if isLoggedIn {
		info, err = read()
		if err != nil {
			return errors.WithStack(err)
		}
	}

	info.Source = string(source)
	info.Username = username
	info.Time = strconv.FormatInt(time.Now().Unix(), 10)

	if err := write(info); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func Clear() error {
	isLoggedIn, err := authFileExists()
	if err != nil {
		return err
	}
	if isLoggedIn {
		filePath, err := authFilePath()
		if err != nil {
			return err
		}
		if err := os.Remove(filePath); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func authFileExists() (bool, error) {
	filePath, err := authFilePath()
	if err != nil {
		return false, errors.WithStack(err)
	}
	return fileutil.Exists(filePath), nil
}

func read() (*Info, error) {
	filePath, err := authFilePath()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	contents, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	info := &Info{}
	if err := json.Unmarshal(contents, info); err != nil {
		return nil, errors.WithStack(err)
	}
	return info, nil
}

func write(info *Info) error {
	contents, err := json.Marshal(info)
	if err != nil {
		return errors.WithStack(err)
	}

	filePath, err := authFilePath()
	if err != nil {
		return errors.WithStack(err)
	}

	err = os.WriteFile(filePath, contents, 0600)
	return errors.WithStack(err)
}

func authFilePath() (string, error) {
	configDir, err := configDir()
	if err != nil {
		return "", errors.WithStack(err)
	}

	return filepath.Join(configDir, "auth.json"), nil
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.WithStack(err)
	}
	return filepath.Join(home, ".config", "devbox"), nil
}
