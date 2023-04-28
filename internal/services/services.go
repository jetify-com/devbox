// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

//lint:file-ignore U1000 Ignore unused function temporarily for debugging
package services

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/a8m/envsubst"
	"github.com/fatih/color"
	"github.com/pkg/errors"

	"go.jetpack.io/devbox/internal/env"
)

type Services map[string]Service

type Service struct {
	Name               string            `json:"name"`
	Env                map[string]string `json:"-"`
	RawPort            string            `json:"port"`
	Start              string            `json:"start"`
	Stop               string            `json:"stop"`
	ProcessComposePath string
}

// TODO: (john) Since moving to process-compose, our services no longer use the old `toggleServices` function. We'll need to clean a lot of this up in a later PR.

type serviceAction int

const (
	startService serviceAction = iota
	stopService
)

func printProxyURL(w io.Writer, services Services) error { // TODO: remove it?
	if !env.IsDevboxCloud() {
		return nil
	}

	hostname, err := os.Hostname()
	if err != nil {
		return errors.WithStack(err)
	}

	printGeneric := false
	for _, service := range services {
		if port, _ := service.Port(); port != "" {
			color.New(color.FgHiGreen).Fprintf(
				w,
				"To access %s on this vm use: %s-%s.svc.devbox.sh\n",
				service.Name,
				hostname,
				port,
			)
		} else {
			printGeneric = true
		}
	}

	if printGeneric {
		color.New(color.FgHiGreen).Fprintf(
			w,
			"To access other services on this vm use: %s-<port>.svc.devbox.sh\n",
			hostname,
		)
	}
	return nil
}

func (s *Service) Port() (string, error) {
	if s.RawPort == "" {
		return "", nil
	}
	return envsubst.String(s.RawPort)
}

func (s *Service) ProcessComposeYaml() (string, bool) {
	return s.ProcessComposePath, true
}

func (s *Service) StartName() string {
	return fmt.Sprintf("%s-service-start", s.Name)
}

func (s *Service) StopName() string {
	return fmt.Sprintf("%s-service-stop", s.Name)
}

func (s *Services) UnmarshalJSON(b []byte) error {
	var m map[string]Service
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	*s = make(Services)
	for name, svc := range m {
		svc.Name = name
		(*s)[name] = svc
	}
	return nil
}
