package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
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
	go func() {
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
	}()

	// We only want events for the specific host.
	return errors.WithStack(watcher.Add(filepath.Join(cloudFilePath(opts.ProjectDir), opts.HostID)))
}

func cloudFilePath(projectDir string) string {
	return filepath.Join(projectDir, ".devbox.cloud")
}

// initCloudDir creates the service status directory and a .gitignore file
func initCloudDir(projectDir, hostID string) error {
	cloudDirPath := cloudFilePath(projectDir)
	_ = os.MkdirAll(filepath.Join(cloudDirPath, hostID), 0755)
	gitignorePath := filepath.Join(cloudDirPath, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		if err := os.WriteFile(
			gitignorePath,
			[]byte("*"),
			0644,
		); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
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
	_ = os.MkdirAll(filepath.Dir(path), 0755) // create path, ignore error
	if err := os.WriteFile(path, content, 0644); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func updateServiceStatusOnRemote(projectDir string, s *ServiceStatus) error {
	if os.Getenv("DEVBOX_REGION") == "" {
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
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	status := &ServiceStatus{}
	return status, errors.WithStack(json.Unmarshal(content, status))
}
