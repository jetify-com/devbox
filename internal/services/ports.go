package services

import (
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/pkg/errors"
)

var disallowedPorts = map[int]string{
	// Anything <= 1024
	1433: "MS-SQL (Microsoft SQL Server database management system)",
	1434: "MS-SQL (Microsoft SQL Server database management system)",
	1521: "Oracle SQL",
	1701: "L2TP (Layer 2 Tunneling Protocol)",
	1723: "PPTP (Point-to-Point Tunneling Protocol)",
	2049: "NFS (Network File System)",
	3000: "Node.js (Server-side JavaScript environment)",
	3001: "Node.js (Server-side JavaScript environment)",
	3306: "MySQL (Database system)",
	3389: "RDP (Remote Desktop Protocol)",
	5060: "SIP (Session Initiation Protocol)",
	5145: "RSH (Remote Shell)",
	5353: "mDNS (Multicast DNS)",
	5432: "PostgreSQL (Database system)",
	5900: "VNC (Virtual Network Computing)",
	6379: "Redis (Database system)",
	8000: "HTTP Alternate (http_alt)",
	8080: "HTTP Alternate (http_alt)",
	8082: "PHP FPM",
	8443: "HTTPS Alternate (https_alt)",
	9443: "Redis Enterprise (Database system)",
}

func getAvailablePort() (int, error) {
	get := func() (int, error) {
		port, err := isPortAvailable(0)
		if err != nil {
			return 0, errors.WithStack(err)
		}
		return port, nil
	}

	for range 1000 {
		port, err := get()
		if err != nil {
			return 0, errors.WithStack(err)
		}

		if isAllowed(port) {
			return port, nil
		}
	}

	return 0, errors.New("no available port")
}

func selectPort(configPort int) (int, error) {
	if configPort != 0 {
		return isPortAvailable(configPort)
	}

	if portStr, exists := os.LookupEnv("DEVBOX_PC_PORT_NUM"); exists {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return 0, fmt.Errorf("invalid DEVBOX_PC_PORT_NUM environment variable: %v", err)
		}
		if port <= 0 {
			return 0, fmt.Errorf("invalid DEVBOX_PC_PORT_NUM environment variable: ports cannot be less than 0")
		}
		return isPortAvailable(port)
	}

	return getAvailablePort()
}

func isAllowed(port int) bool {
	return port > 1024 && disallowedPorts[port] == ""
}

func isPortAvailable(port int) (int, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return 0, fmt.Errorf("port %d is already in use", port)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}
