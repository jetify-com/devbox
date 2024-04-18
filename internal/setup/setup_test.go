package setup

import (
	"context"
	"errors"
	"testing"
)

type testTask struct {
	RunFunc      func(ctx context.Context) error
	NeedsRunFunc func(ctx context.Context, lastRun RunInfo) bool
}

func (t *testTask) Run(ctx context.Context) error {
	return t.RunFunc(ctx)
}

func (t *testTask) NeedsRun(ctx context.Context, lastRun RunInfo) bool {
	return t.NeedsRunFunc(ctx, lastRun)
}

func TestTaskNeedsRunTrue(t *testing.T) {
	tempXDGStateDir(t)

	ran := false
	task := &testTask{
		RunFunc: func(ctx context.Context) error {
			ran = true
			return nil
		},
		NeedsRunFunc: func(context.Context, RunInfo) bool {
			return true
		},
	}

	err := Run(context.Background(), t.Name(), task)
	if err != nil {
		t.Error("got non-nil error:", err)
	}
	if !ran {
		t.Error("got ran = false, want true")
	}
}

func TestTaskNeedsRunFalse(t *testing.T) {
	tempXDGStateDir(t)

	ran := false
	task := &testTask{
		RunFunc: func(ctx context.Context) error {
			ran = true
			return nil
		},
		NeedsRunFunc: func(context.Context, RunInfo) bool {
			return false
		},
	}

	err := Run(context.Background(), t.Name(), task)
	if err != nil {
		t.Error("got non-nil error:", err)
	}
	if ran {
		t.Error("got ran = true, want false")
	}
}

func TestTaskLastRun(t *testing.T) {
	tempXDGStateDir(t)

	task := &testTask{
		RunFunc:      func(ctx context.Context) error { return nil },
		NeedsRunFunc: func(context.Context, RunInfo) bool { return true },
	}
	err := Run(context.Background(), t.Name(), task)
	if err != nil {
		t.Error("got non-nil error on first run:", err)
	}

	task.NeedsRunFunc = func(ctx context.Context, lastRun RunInfo) bool {
		if lastRun.Time.IsZero() {
			t.Error("got zero lastRun.Time on second run")
		}
		if lastRun.Version == "" {
			t.Error("got empty lastRun.Version on second run")
		}
		if lastRun.Error != "" {
			t.Errorf("got non-empty lastRun.Error on second run: %v", lastRun.Error)
		}
		return false
	}
	err = Run(context.Background(), t.Name(), task)
	if err != nil {
		t.Error("got non-nil error on second run:", err)
	}
}

func TestTaskConfirmPromptAllow(t *testing.T) {
	tempXDGStateDir(t)

	task := &testTask{
		RunFunc:      func(ctx context.Context) error { return nil },
		NeedsRunFunc: func(context.Context, RunInfo) bool { return true },
	}

	defaultPrompt = func(string) (response any, err error) { return true, nil }
	err := ConfirmRun(context.Background(), t.Name(), task, "continue?")
	if err != nil {
		t.Error("got non-nil error:", err)
	}
}

func TestTaskConfirmPromptDeny(t *testing.T) {
	tempXDGStateDir(t)

	task := &testTask{
		RunFunc:      func(ctx context.Context) error { return nil },
		NeedsRunFunc: func(context.Context, RunInfo) bool { return true },
	}

	defaultPrompt = func(string) (response any, err error) { return false, nil }
	err := ConfirmRun(context.Background(), t.Name(), task, "continue?")
	if err == nil {
		t.Error("got nil error, want ErrUserRefused")
	} else if !errors.Is(err, ErrUserRefused) {
		t.Error("got errors.Is(err, ErrUserRefused) == false for error:", err)
	}
}

func tempXDGStateDir(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
}
