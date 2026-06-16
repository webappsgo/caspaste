// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package portutil

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
)

// FindUnusedPort finds an unused port in the specified range
// Returns the first available port found or an error if none available
func FindUnusedPort(minPort, maxPort int) (int, error) {
	if minPort < 1 || maxPort > 65535 || minPort > maxPort {
		return 0, fmt.Errorf("invalid port range: %d-%d", minPort, maxPort)
	}

	// Try random ports in range until one works
	for attempts := 0; attempts < 100; attempts++ {
		port := rand.Intn(maxPort-minPort+1) + minPort
		if IsPortAvailable(port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available port found in range %d-%d after 100 attempts", minPort, maxPort)
}

// IsPortAvailable checks if a port is available for binding
func IsPortAvailable(port int) bool {
	if port < 1 || port > 65535 {
		return false
	}

	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// ParsePorts parses port configuration string
// Formats supported:
//   "8080"        → httpPort=8080, httpsPort=0
//   "8080,64453"  → httpPort=8080, httpsPort=64453
func ParsePorts(portStr string) (httpPort, httpsPort int, err error) {
	if portStr == "" {
		return 0, 0, fmt.Errorf("port string is empty")
	}

	parts := strings.Split(portStr, ",")

	// Parse HTTP port (first port)
	httpPort, err = strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid HTTP port: %w", err)
	}
	if httpPort < 1 || httpPort > 65535 {
		return 0, 0, fmt.Errorf("HTTP port out of range: %d", httpPort)
	}

	// Parse HTTPS port (second port, if present)
	if len(parts) == 2 {
		httpsPort, err = strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid HTTPS port: %w", err)
		}
		if httpsPort < 1 || httpsPort > 65535 {
			return 0, 0, fmt.Errorf("HTTPS port out of range: %d", httpsPort)
		}
	}

	return httpPort, httpsPort, nil
}

// FormatPorts formats ports back to string
// Single port: "8080"
// Dual ports: "8080,64453"
func FormatPorts(httpPort, httpsPort int) string {
	if httpsPort > 0 {
		return fmt.Sprintf("%d,%d", httpPort, httpsPort)
	}
	return strconv.Itoa(httpPort)
}
