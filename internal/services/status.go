package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/debug"
)

// statusUpdate returns a possibly updated status file and a boolean indicating
// whether the status file should be saved. This prevents infinite loops when
// the status file is updated.
type statusUpdate func(status StatusFile) (StatusFile, bool)

func ListenToChanges(ctx context.Context, w io.Writer, projectDir string, update statusUpdate) error {
	if err := initStatusFile(projectDir); err != nil {
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
		debug.Log("listening to changes on %s", statusFilePath(projectDir))
		for {
			select {
			case event, ok := <-watcher.Events:
				debug.Log("event: %s", event)
				if !ok {
					return
				}
				if os.Getenv("DEVBOX_REGION") != "" {
					status, _ := readStatusFile(projectDir)
					fmt.Fprintf(w, "Status changed: %v", status.Hosts["6e8292db74ee87"].Services["apache"])
				}
				// mutagen sync changes show up as create events
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Chmod) {
					debug.Log("file changed (write event)")
					status, err := readStatusFile(projectDir)
					if err != nil {
						fmt.Fprintf(w, "Error reading status file: %s\n", err)
						continue
					}
					// Only call callback if something has changed
					debug.Log("status changed, calling callback")
					status, saveChanges := update(status)
					if saveChanges {
						if err := updateStatusFile(projectDir, status); err != nil {
							fmt.Fprintf(w, "Error updating status file: %s\n", err)
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Fprintf(w, "error: %s\n", err)
			}
		}
	}()

	return errors.WithStack(watcher.Add(statusFilePath(projectDir)))
}

func cloudFilePath(projectDir string) string {
	return filepath.Join(projectDir, ".devbox.cloud")
}

func statusFilePath(projectDir string) string {
	return filepath.Join(cloudFilePath(projectDir), "services.json")
}

// initStatusFile creates the status file if it doesn't exist.
func initStatusFile(projectDir string) error {
	filePath := statusFilePath(projectDir)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		sf := StatusFile{Hosts: map[string]*host{}}
		content, err := json.MarshalIndent(sf, "", "  ")
		if err != nil {
			return errors.WithStack(err)
		}
		_ = os.Mkdir(cloudFilePath(projectDir), 0755)
		if err := os.WriteFile(
			filepath.Join(cloudFilePath(projectDir), ".gitignore"),
			[]byte("*"),
			0644,
		); err != nil {
			return errors.WithStack(err)
		}

		if err := os.WriteFile(filePath, content, 0644); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

type StatusFile struct {
	Hosts map[string]*host `json:"hosts"`
}

type host struct {
	Services map[string]*serviceStatus `json:"services"`
}

type serviceStatus struct {
	LocalPort string `json:"local_port"`
	Name      string `json:"name"`
	Port      string `json:"port"`
	Running   bool   `json:"running"`
}

func updateStatusFile(projectDir string, status StatusFile) error {
	content, err := json.Marshal(status)
	if err != nil {
		return errors.WithStack(err)
	}
	if err := os.WriteFile(statusFilePath(projectDir), content, 0644); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func updateServiceStatus(projectDir string, statusUpdate *serviceStatus) error {
	if os.Getenv("DEVBOX_REGION") == "" {
		return nil
	}
	hostname, err := hostname()
	if err != nil {
		return errors.WithStack(err)
	}

	status, err := readStatusFile(projectDir)
	if err != nil {
		return err
	}

	if _, ok := status.Hosts[hostname]; !ok {
		status.Hosts[hostname] = &host{Services: map[string]*serviceStatus{}}
	}

	status.Hosts[hostname].Services[statusUpdate.Name] = statusUpdate
	return updateStatusFile(projectDir, status)
}

func readStatusFile(projectDir string) (StatusFile, error) {
	if err := initStatusFile(projectDir); err != nil {
		return StatusFile{}, err
	}

	content, err := os.ReadFile(statusFilePath(projectDir))
	if err != nil {
		return StatusFile{}, errors.WithStack(err)
	}
	var status StatusFile
	if err := json.Unmarshal(content, &status); err != nil {
		return StatusFile{}, errors.WithStack(err)
	}
	return status, nil
}

func hostname() (string, error) {
	stdout, err := exec.Command("uname", "-n").Output()
	if err != nil {
		return "", errors.WithStack(err)
	}
	return strings.TrimSpace(string(stdout)), nil
}
