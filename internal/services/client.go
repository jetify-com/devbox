// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/f1bonacc1/process-compose/src/types"
)

type processStates = types.ProcessesState

type Process struct {
	Name     string
	Status   string
	ExitCode int
}

func StartServices(ctx context.Context, w io.Writer, serviceName, projectDir string) error {
	path := fmt.Sprintf("/process/start/%s", serviceName)

	body, status, err := clientRequest(path, http.MethodPost, 0, projectDir)
	if err != nil {
		return err
	}

	switch status {
	case http.StatusOK:
		fmt.Fprintf(w, "Service %s started.\n", serviceName)
		return nil
	default:
		return fmt.Errorf("error starting service %s: %s", serviceName, body)
	}
}

func StopServices(ctx context.Context, serviceName, projectDir string, w io.Writer) error {
	path := fmt.Sprintf("/process/stop/%s", serviceName)

	body, status, err := clientRequest(path, http.MethodPatch, 0, projectDir)
	if err != nil {
		return err
	}

	switch status {
	case http.StatusOK:
		fmt.Fprintf(w, "Service %s stopped.\n", serviceName)
		return nil
	default:
		return fmt.Errorf("error stopping service %s: %s", serviceName, body)
	}
}

func RestartServices(ctx context.Context, serviceName, projectDir string, w io.Writer) error {
	path := fmt.Sprintf("/process/restart/%s", serviceName)

	body, status, err := clientRequest(path, http.MethodPost, 0, projectDir)
	if err != nil {
		return err
	}

	switch status {
	case http.StatusOK:
		fmt.Fprintf(w, "Service %s restarted.\n", serviceName)
		return nil
	default:
		return fmt.Errorf("error restarting service %s: %s", serviceName, body)
	}
}

func ListServices(ctx context.Context, projectDir string, port int, w io.Writer) ([]Process, error) {
	path := "/processes"
	results := []Process{}

	body, status, err := clientRequest(path, http.MethodGet, port, projectDir)
	if err != nil {
		return results, err
	}

	switch status {
	case http.StatusOK:
		var processes processStates
		err := json.Unmarshal([]byte(body), &processes)
		if err != nil {
			return results, err
		}
		for _, process := range processes.States {
			results = append(results, Process{
				Name:     process.Name,
				Status:   process.Status,
				ExitCode: process.ExitCode,
			})
		}
		return results, nil
	default:
		return results, fmt.Errorf("unable to list services: %s", body)
	}
}

func clientRequest(path string, method string, port int, projectDir string) (string, int, error) {
	var err error
	if port == 0 {
		port, err = GetProcessManagerPort(projectDir)
	}
	if err != nil {
		err := fmt.Errorf("unable to connect to process-compose server: %s", err.Error())
		return "", 0, err
	}
	req, err := http.NewRequest(method, fmt.Sprintf("http://localhost:%d%s", port, path), nil)
	if err != nil {
		return "", 0, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}

	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return "", 0, err
	}
	body := buf.String()
	status := resp.StatusCode

	return body, status, nil
}
