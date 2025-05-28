package services

import (
	"fmt"
	"net"
	"time"
)

// FindAvailablePort finds an available port starting from the given port
func FindAvailablePort(startPort int) (int, error) {
	for port := startPort; port < startPort+100; port++ {
		if isPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port found in range %d-%d", startPort, startPort+99)
}

// isPortAvailable checks if a port is available for binding
func isPortAvailable(port int) bool {
	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return false
	}
	defer listener.Close()
	return true
}

// GetAvailablePortInRange finds an available port in a specific range
func GetAvailablePortInRange(startPort, endPort int) (int, error) {
	for port := startPort; port <= endPort; port++ {
		if isPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port found in range %d-%d", startPort, endPort)
}

// WaitForPortToBecomeAvailable waits for a port to become available with timeout
func WaitForPortToBecomeAvailable(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if isPortAvailable(port) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("port %d did not become available within %v", port, timeout)
}