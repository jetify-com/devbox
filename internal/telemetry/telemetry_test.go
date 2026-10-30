package telemetry

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// TestErrorBasic does a very simple sanity check to ensure the error can be sent
// to the sentry and segment buffers
func TestErrorBasic(t *testing.T) {
	setTestStateDir(t)
	started = true

	fakeErr := errors.New("fake error")
	meta := Metadata{}

	Error(fakeErr, meta)
	if !uploadNow.Load() {
		t.Error("Error didn't request an immediate upload")
	}
}

func setTestStateDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	segmentBufferDir = filepath.Join(dir, "segment")
	sentryBufferDir = filepath.Join(dir, "sentry")
	deviceIDFile = filepath.Join(dir, "telemetry", "device-id")
	lastUploadFile = filepath.Join(dir, "telemetry", "last-upload")
	uploadNow.Store(false)
	t.Cleanup(func() { uploadNow.Store(false) })
}

func TestLoadDeviceIDCached(t *testing.T) {
	setTestStateDir(t)

	if err := writeStateFile(deviceIDFile, []byte("cached-id\n")); err != nil {
		t.Fatal(err)
	}
	if got := loadDeviceID(); got != "cached-id" {
		t.Errorf("got device ID %q, want %q", got, "cached-id")
	}
}

func TestLoadDeviceIDWritesCache(t *testing.T) {
	setTestStateDir(t)

	id := loadDeviceID()
	if id == "" {
		t.Skip("machine ID is unavailable on this machine")
	}
	data, err := os.ReadFile(deviceIDFile)
	if err != nil {
		t.Fatalf("device ID wasn't cached: %v", err)
	}
	if string(data) != id {
		t.Errorf("got cached device ID %q, want %q", data, id)
	}
}

func TestShouldUpload(t *testing.T) {
	writeLastUpload := func(t *testing.T, ts time.Time) {
		t.Helper()
		data := []byte(strconv.FormatInt(ts.Unix(), 10))
		if err := writeStateFile(lastUploadFile, data); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("NeverUploaded", func(t *testing.T) {
		setTestStateDir(t)
		if !shouldUpload() {
			t.Error("got shouldUpload() = false, want true")
		}
	})
	t.Run("RecentUpload", func(t *testing.T) {
		setTestStateDir(t)
		writeLastUpload(t, time.Now())
		if shouldUpload() {
			t.Error("got shouldUpload() = true, want false")
		}
	})
	t.Run("StaleUpload", func(t *testing.T) {
		setTestStateDir(t)
		writeLastUpload(t, time.Now().Add(-uploadInterval))
		if !shouldUpload() {
			t.Error("got shouldUpload() = false, want true")
		}
	})
	t.Run("FutureUpload", func(t *testing.T) {
		setTestStateDir(t)
		writeLastUpload(t, time.Now().Add(time.Hour))
		if !shouldUpload() {
			t.Error("got shouldUpload() = false, want true")
		}
	})
	t.Run("Error", func(t *testing.T) {
		setTestStateDir(t)
		writeLastUpload(t, time.Now())
		uploadNow.Store(true)
		if !shouldUpload() {
			t.Error("got shouldUpload() = false, want true")
		}
	})
	t.Run("ManyBufferedEvents", func(t *testing.T) {
		setTestStateDir(t)
		writeLastUpload(t, time.Now())
		for i := range uploadThreshold {
			file := filepath.Join(segmentBufferDir, strconv.Itoa(i)+".json")
			if err := writeStateFile(file, []byte("{}")); err != nil {
				t.Fatal(err)
			}
		}
		if !shouldUpload() {
			t.Error("got shouldUpload() = false, want true")
		}
	})
}
