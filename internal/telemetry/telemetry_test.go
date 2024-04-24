package telemetry

import (
	"errors"
	"testing"
)

// TestErrorBasic does a very simple sanity check to ensure the error can be sent
// to the sentry and segment buffers
func TestErrorBasic(t *testing.T) {
	segmentBufferDir = t.TempDir()
	sentryBufferDir = t.TempDir()
	started = true

	fakeErr := errors.New("fake error")
	meta := Metadata{}

	Error(fakeErr, meta)
}
