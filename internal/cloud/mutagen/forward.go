package mutagen

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
)

func ForwardCreate(host, localPort, remotePort string) (string, error) {
	var err error
	if localPort == "" {
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
	args := []string{"forward", "create", local, remote, "--label", "devbox=true"}
	return localPort, execMutagen(args)
}

func ForwardTerminateAll() error {
	args := []string{"forward", "terminate", "--label-selector", "devbox=true"}
	return execMutagen(args)
}

func ForwardList() ([]string, error) {
	args := []string{"forward", "list", "--label-selector", "devbox=true", "--template", "{{json .}}"}
	out, err := execMutagenOut(args, nil)
	if err != nil {
		return nil, err
	}

	list := []struct {
		Source struct {
			Connected bool   `json:"connected"`
			Endpoint  string `json:"endpoint"`
		} `json:"source"`
		Destination struct {
			Endpoint string `json:"endpoint"`
		} `json:"destination"`
		LastError string `json:"lastError"`
	}{}

	if err := json.Unmarshal(out, &list); err != nil {
		return nil, errors.WithStack(err)
	}

	result := []string{}
	for _, item := range list {
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
