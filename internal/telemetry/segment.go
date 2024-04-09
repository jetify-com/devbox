// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package telemetry

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	segment "github.com/segmentio/analytics-go"
	"go.jetpack.io/devbox/internal/nix"

	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/devbox/internal/envir"
)

var segmentClient segment.Client

func initSegmentClient() bool {
	if build.TelemetryKey == "" {
		return false
	}

	var err error
	segmentClient, err = segment.NewWithConfig(build.TelemetryKey, segment.Config{
		Logger:  segment.StdLogger(log.New(io.Discard, "", 0)),
		Verbose: false,
	})
	return err == nil
}

func newTrackMessage(name string, meta Metadata) *segment.Track {
	nixVersion, err := nix.Version()
	if err != nil {
		nixVersion = "unknown"
	}

	dur := time.Since(procStartTime)
	if !meta.CommandStart.IsZero() {
		dur = time.Since(meta.CommandStart)
	}
	return &segment.Track{
		MessageId:   newEventID(),
		Type:        "track",
		AnonymousId: deviceID,
		UserId:      userID,
		Timestamp:   time.Now(),
		Event:       name,
		Context: &segment.Context{
			Device: segment.DeviceInfo{
				Id: deviceID,
			},
			App: segment.AppInfo{
				Name:    appName,
				Version: build.Version,
			},
			OS: segment.OSInfo{
				Name: build.OS(),
			},
		},
		Properties: segment.Properties{
			"cloud_region": meta.CloudRegion,
			"command":      meta.Command,
			"command_args": meta.CommandFlags,
			"duration":     dur.Milliseconds(),
			"packages":     meta.Packages,
			"shell":        os.Getenv(envir.Shell),
			"shell_access": shellAccess(),
			"nix_version":  nixVersion,
		},
	}
}

// bufferSegmentMessage buffers a Segment message to disk so that Report can
// upload it later.
func bufferSegmentMessage(id string, msg segment.Message) {
	bufferEvent(filepath.Join(segmentBufferDir, id+".json"), msg)
}

type shellAccessKind string

const (
	local   shellAccessKind = "local"
	ssh     shellAccessKind = "ssh"
	browser shellAccessKind = "browser"
)

func shellAccess() shellAccessKind {
	// Check if running in devbox cloud
	if envir.IsDevboxCloud() {
		// Check if running via ssh tty (i.e. ssh shell)
		if os.Getenv(envir.SSHTTY) != "" {
			return ssh
		}
		return browser
	}
	return local
}
