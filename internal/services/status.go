// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

//lint:ignore U1000 Ignore unused function temporarily for debugging

package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"

	"go.jetpack.io/devbox/internal/envir"
)

// updateFunc returns a possibly updated service status and a boolean indicating
// whether the status file should be saved. This prevents infinite loops when
// the status file is updated.
type updateFunc func(status *ServiceStatus) (*ServiceStatus, bool)

type ListenerOpts struct {
	HostID     string
	ProjectDir string
	UpdateFunc updateFunc
	Writer     io.Writer
}

func ListenToChanges(ctx context.Context, opts *ListenerOpts) error {
	if err := initCloudDir(opts.ProjectDir, opts.HostID); err != nil {
		return err
	}

	// Create new watcher.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.WithStack(err)
	}

	go func() {
		<-ctx.Done()
		watcher.Close()
	}()

	// Start listening for events.
	go listenToEvents(watcher, opts)

	// We only want events for the specific host.
	return errors.WithStack(watcher.Add(filepath.Join(cloudFilePath(opts.ProjectDir), opts.HostID)))
}

func listenToEvents(watcher *fsnotify.Watcher, opts *ListenerOpts) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// mutagen sync changes show up as create events
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				status, err := readServiceStatus(event.Name)
				if err != nil {
					fmt.Fprintf(opts.Writer, "Error reading status file: %s\n", err)
					continue
				}

				status, saveChanges := opts.UpdateFunc(status)
				if saveChanges {
					if err := writeServiceStatusFile(event.Name, status); err != nil {
						fmt.Fprintf(opts.Writer, "Error updating status file: %s\n", err)
					}
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(opts.Writer, "error: %s\n", err)
		}
	}
}

func cloudFilePath(projectDir string) string {
	return filepath.Join(projectDir, ".devbox.cloud")
}

// initCloudDir creates the service status directory and a .gitignore file
func initCloudDir(projectDir, hostID string) error {
	cloudDirPath := cloudFilePath(projectDir)
	_ = os.MkdirAll(filepath.Join(cloudDirPath, hostID), 0o755)
	gitignorePath := filepath.Join(cloudDirPath, ".gitignore")
	_, err := os.Stat(gitignorePath)
	if !errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return errors.WithStack(os.WriteFile(gitignorePath, []byte("*"), 0o644))
}

type ServiceStatus struct {
	LocalPort string `json:"local_port"`
	Name      string `json:"name"`
	Port      string `json:"port"`
	Running   bool   `json:"running"`
}

func writeServiceStatusFile(path string, status *ServiceStatus) error {
	content, err := json.Marshal(status)
	if err != nil {
		return errors.WithStack(err)
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755) // create path, ignore error
	return errors.WithStack(os.WriteFile(path, content, 0o644))
}

//lint:ignore U1000 Ignore unused function temporarily for debugging
func updateServiceStatusOnRemote(projectDir string, s *ServiceStatus) error {
	if !envir.IsDevboxCloud() {
		return nil
	}
	host, err := os.Hostname()
	if err != nil {
		return errors.WithStack(err)
	}

	cloudDirPath := cloudFilePath(projectDir)
	return writeServiceStatusFile(filepath.Join(cloudDirPath, host, s.Name+".json"), s)
}

func readServiceStatus(path string) (*ServiceStatus, error) {
	_, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	status := &ServiceStatus{}
	return status, errors.WithStack(json.Unmarshal(content, status))
}
