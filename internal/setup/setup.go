// Package setup performs setup tasks and records metadata about when they're
// run.
package setup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/redact"
	"go.jetpack.io/devbox/internal/xdg"
)

// ErrUserRefused indicates that the user responded no to a confirmation prompt.
var ErrUserRefused = errors.New("user refused run")

// Task is a setup action that can conditionally run based on the state of a
// previous run.
type Task interface {
	Run(ctx context.Context) error

	// NeedsRun returns true if the task needs to be run. It should assume
	// that lastRun persists across executions of the program and is unique
	// for each user.
	//
	// A task that should only run once can check if lastRun.Time is the zero value.
	// A task that only runs after an update can check if lastRun.Version < build.Version.
	// A retryable task can check lastRun.Error to see if the previous run failed.
	NeedsRun(ctx context.Context, lastRun RunInfo) bool
}

// RunInfo contains metadata that describes the most recent run of a task.
type RunInfo struct {
	// Time is the last time the task ran.
	Time time.Time `json:"time"`

	// Version is the version of Devbox that last ran the task.
	Version string `json:"version"`

	// Error is the error message returned by the last run. It's empty if
	// the last run succeeded.
	Error string `json:"error,omitempty"`
}

// Run runs a setup task and stores its state under a given key. Keys are
// namespaced by user. It only calls the task's Run method when NeedsRun returns
// true.
func Run(ctx context.Context, key string, task Task) error {
	return run(ctx, key, task, "")
}

// Run interactively prompts the user to confirm that it's ok to run a setup
// task. It only prompts the user if the task's NeedsRun method returns true. If
// the user refuses to run the task, then ConfirmPrompt will not ask them again.
// Call [Reset] to reset the task's state and re-prompt the user.
func ConfirmRun(ctx context.Context, key string, task Task, prompt string) error {
	if prompt == "" {
		return taskError(key, redact.Errorf("empty confirmation prompt"))
	}
	return run(ctx, key, task, prompt)
}

var defaultPrompt = func(msg string) (response any, err error) {
	err = survey.AskOne(&survey.Confirm{Message: msg}, &response)
	return response, err
}

func run(ctx context.Context, key string, task Task, prompt string) error {
	state := loadState(key)
	if !task.NeedsRun(ctx, state.LastRun) {
		return nil
	}

	oldState, newState := state, &state
	defer func() {
		if oldState != *newState {
			saveState(key, *newState)
		}
	}()

	if prompt != "" {
		state.ConfirmPrompt.Message = prompt
		if state.ConfirmPrompt.Asked && !state.ConfirmPrompt.Allowed {
			// We've asked before and the user said no.
			return taskError(key, ErrUserRefused)
		}

		resp, err := defaultPrompt(prompt)
		if err != nil {
			return taskError(key, redact.Errorf("prompt for confirmation: %v", err))
		}
		state.ConfirmPrompt.Asked = true
		state.ConfirmPrompt.Allowed, _ = resp.(bool)
		if !state.ConfirmPrompt.Allowed {
			return taskError(key, ErrUserRefused)
		}
	}

	state.LastRun = RunInfo{
		Time:    time.Now(),
		Version: build.Version,
	}
	if err := task.Run(ctx); err != nil {
		state.LastRun.Error = err.Error()
		return taskError(key, err)
	}
	return nil
}

// Reset removes a task's state so that it acts as if it has never run.
func Reset(key string) {
	err := os.Remove(statePath(key))
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		err = taskError(key, fmt.Errorf("remove state file: %v", err))
		debug.Log(err.Error())
	}
}

type state struct {
	ConfirmPrompt confirmPrompt `json:"confirm_prompt,omitempty"`
	LastRun       RunInfo       `json:"last_run,omitempty"`
}

type confirmPrompt struct {
	Message string `json:"message"`
	Asked   bool   `json:"asked"`
	Allowed bool   `json:"allowed"`
}

func loadState(key string) state {
	path := statePath(key)
	b, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			err = taskError(key, fmt.Errorf("load state file: %v", err))
			debug.Log(err.Error())
		}
		return state{}
	}
	loaded := state{}
	if err := json.Unmarshal(b, &loaded); err != nil {
		err = taskError(key, fmt.Errorf("load state file %s: %v", path, err))
		debug.Log(err.Error())
		return state{}
	}
	return loaded
}

func saveState(key string, s state) {
	path := statePath(key)
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		err = taskError(key, fmt.Errorf("save state file: %v", err))
		debug.Log(err.Error())
		return
	}

	err = os.WriteFile(path, data, 0o644)
	if errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(filepath.Dir(path), 0o755)
		if err == nil {
			err = os.WriteFile(path, data, 0o644)
		}
	}
	if err != nil {
		err = taskError(key, fmt.Errorf("save state file: %v", err))
		debug.Log(err.Error())
		return
	}

	sudoUID, sudoGID := os.Getenv("SUDO_UID"), os.Getenv("SUDO_GID")
	if sudoUID != "" || sudoGID != "" {
		uid, err := strconv.Atoi(sudoUID)
		if err != nil {
			uid = -1
		}
		gid, err := strconv.Atoi(sudoGID)
		if err != nil {
			gid = -1
		}
		err = os.Chown(path, uid, gid)
		if err != nil {
			err = taskError(key, fmt.Errorf("chown state file to non-sudo user: %v", err))
			debug.Log(err.Error())
		}
	}
}

func statePath(key string) string {
	dir := xdg.StateSubpath("devbox")
	name := strings.ReplaceAll(key, "/", "-")
	return filepath.Join(dir, name)
}

func taskError(key string, err error) error {
	if err == nil {
		return nil
	}
	return redact.Errorf("setup: task %s: %w", key, err)
}
