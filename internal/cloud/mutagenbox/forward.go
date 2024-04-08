// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package mutagenbox

import (
	"fmt"
	"net"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cloud/mutagen"
)

func ForwardCreate(host, localPort, remotePort string) (string, error) {
	var err error
	if localPort == "" || localPort == "0" {
		localPort, err = getFreePort()
		if err != nil {
			return "", err
		}
	}

	if !isPortAvailable(localPort) {
		return "", usererr.New("Port %s is not available", localPort)
	}

	local := "tcp:127.0.0.1:" + localPort
	remote := host + ":22:tcp::" + remotePort
	labels := map[string]string{
		"devbox":      "true",
		"remote-host": host,
		"remote-port": remotePort,
	}
	env, err := DefaultEnv()
	if err != nil {
		return "", err
	}
	return localPort, mutagen.ForwardCreate(env, local, remote, labels)
}

func ForwardCreateIfNotExists(host, localPort, remotePort string) (string, error) {
	forwards, err := forwardListWithLabels(map[string]string{
		"remote-host": host,
		"remote-port": remotePort,
	})
	if err != nil {
		return "", err
	}
	if len(forwards) > 0 {
		srcParts := strings.Split(forwards[0].Source.Endpoint, ":")
		return srcParts[len(srcParts)-1], nil
	}
	return ForwardCreate(host, localPort, remotePort)
}

func ForwardTerminateAll() error {
	env, err := DefaultEnv()
	if err != nil {
		return err
	}
	return mutagen.ForwardTerminate(env, map[string]string{"devbox": "true"})
}

func ForwardTerminateByHost(host string) error {
	env, err := DefaultEnv()
	if err != nil {
		return err
	}
	return mutagen.ForwardTerminate(env, map[string]string{
		"devbox":      "true",
		"remote-host": host,
	})
}

func ForwardTerminateByHostPort(host, port string) error {
	env, err := DefaultEnv()
	if err != nil {
		return err
	}
	return mutagen.ForwardTerminate(env, map[string]string{
		"devbox":      "true",
		"remote-host": host,
		"remote-port": port,
	})
}

func ForwardList() ([]string, error) {
	forwards, err := forwardListWithLabels(map[string]string{})
	if err != nil {
		return nil, err
	}

	result := []string{}
	for _, item := range forwards {
		srcParts := strings.Split(item.Source.Endpoint, ":")
		destParts := strings.Split(item.Destination.Endpoint, ":")
		result = append(result, fmt.Sprintf(
			"%s:%s connected: %t %s",
			srcParts[len(srcParts)-1],
			destParts[len(destParts)-1],
			item.Source.Connected,
			lo.Ternary(item.LastError != "", "Error: "+item.LastError, ""),
		))
	}

	return result, nil
}

func forwardListWithLabels(labels map[string]string) ([]mutagen.Forward, error) {
	env, err := DefaultEnv()
	if err != nil {
		return nil, err
	}
	labels["devbox"] = "true" // Add this to all labels
	return mutagen.ForwardList(env, labels)
}

func isPortAvailable(port string) bool {
	ln, err := net.Listen("tcp", net.JoinHostPort("localhost", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

func getFreePort() (string, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return "", errors.WithStack(err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return "", errors.WithStack(err)
	}
	defer l.Close()
	return fmt.Sprintf("%d", l.Addr().(*net.TCPAddr).Port), nil
}
