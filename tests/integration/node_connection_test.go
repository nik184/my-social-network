package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ConnectionInfoResponse represents the API response for connection info
type ConnectionInfoResponse struct {
	PeerID         string   `json:"peerId"`
	PublicAddress  string   `json:"publicAddress,omitempty"`
	Port           int      `json:"port,omitempty"`
	LocalAddresses []string `json:"localAddresses"`
	IsPublicNode   bool     `json:"isPublicNode"`
}

// PeerInfoResponse represents peer information in API responses
type PeerInfoResponse struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Addresses      []string  `json:"addresses"`
	FirstSeen      time.Time `json:"first_seen"`
	LastSeen       time.Time `json:"last_seen"`
	IsValidated    bool      `json:"is_validated"`
	ConnectionType string    `json:"connection_type"`
	HasAvatar      bool      `json:"has_avatar"`
}

// SecondDegreePeerResponse represents a second-degree peer in API responses
type SecondDegreePeerResponse struct {
	PeerID      string `json:"peer_id"`
	PeerName    string `json:"peer_name"`
	ViaPeerID   string `json:"via_peer_id"`
	ViaPeerName string `json:"via_peer_name"`
}

// TestTwoIsolatedNodesConnection tests the ability of two isolated nodes to start and establish a connection
// This test verifies:
// 1. Two separate containers can be started simulating isolated nodes
// 2. The nodes can discover each other's connection information
// 3. The nodes can successfully establish a P2P connection
// 4. The connection is bidirectional and both nodes recognize each other
func TestTwoIsolatedNodesConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping isolated nodes connection test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	t.Log("🚀 Starting two isolated node connection test...")

	// Build application image using existing Dockerfile
	dockerfile := getDockerfileContent()

	// Start first isolated node (Node A)
	t.Log("📦 Starting Node A container...")
	nodeA, err := startIsolatedNode(ctx, "node-a", dockerfile)
	require.NoError(t, err, "Failed to start Node A container")
	defer func() {
		if err := nodeA.Terminate(ctx); err != nil {
			t.Logf("Warning: Failed to terminate Node A: %v", err)
		}
	}()

	// Start second isolated node (Node B)
	t.Log("📦 Starting Node B container...")
	nodeB, err := startIsolatedNode(ctx, "node-b", dockerfile)
	require.NoError(t, err, "Failed to start Node B container")
	defer func() {
		if err := nodeB.Terminate(ctx); err != nil {
			t.Logf("Warning: Failed to terminate Node B: %v", err)
		}
	}()

	// Wait for applications to fully initialize
	t.Log("⏳ Waiting for nodes to initialize...")
	time.Sleep(15 * time.Second)

	// Debug: Check container logs for more details
	logs, err := nodeA.Logs(ctx)
	if err == nil {
		// Read all available logs
		logBytes := make([]byte, 16384)
		n, _ := logs.Read(logBytes)
		fullLogs := string(logBytes[:n])
		t.Logf("Node A full logs (%d bytes):\n%s", n, fullLogs)

		// Check if there are any obvious error patterns
		if strings.Contains(fullLogs, "error") || strings.Contains(fullLogs, "Error") ||
			strings.Contains(fullLogs, "failed") || strings.Contains(fullLogs, "Failed") {
			t.Logf("⚠️  Detected potential errors in Node A logs")
		}
	}

	// Debug: Check what ports are actually exposed
	ports, err := nodeA.Ports(ctx)
	if err == nil {
		t.Logf("Node A exposed ports: %v", ports)
	}

	// Get connection information from Node A
	t.Log("🔍 Retrieving connection info from Node A...")
	nodeAInfo, err := getNodeConnectionInfo(ctx, nodeA)
	require.NoError(t, err, "Failed to get Node A connection info")
	t.Logf("✅ Node A - PeerID: %s, Addresses: %d", nodeAInfo.PeerID, len(nodeAInfo.LocalAddresses))

	// Get Node A's container IP for direct connection
	nodeAIP, err := nodeA.ContainerIP(ctx)
	require.NoError(t, err, "Failed to get Node A container IP")
	t.Logf("🌐 Node A container IP: %s", nodeAIP)

	// Attempt to connect Node B to Node A
	t.Log("🔗 Connecting Node B to Node A...")
	err = connectNodeToTarget(ctx, nodeB, nodeAIP, 9000, nodeAInfo.PeerID)
	require.NoError(t, err, "Node B should be able to connect to Node A")
	t.Log("✅ Connection request sent from Node B to Node A")

	// Wait for connection to establish and stabilize
	t.Log("⏳ Waiting for connection to establish...")
	time.Sleep(8 * time.Second)

	// Verify Node B has connected to Node A
	t.Log("🔍 Verifying Node B's connections...")
	nodeBPeers, err := getNodePeerInfo(ctx, nodeB)
	require.NoError(t, err, "Failed to get Node B peer info")
	assert.GreaterOrEqual(t, len(nodeBPeers), 1, "Node B should have at least 1 connected peer")

	// Check if Node A is in Node B's peer list
	var nodeAFoundInB bool
	for _, peer := range nodeBPeers {
		if peer.ID == nodeAInfo.PeerID && peer.Name == "node-a" {
			nodeAFoundInB = true
			t.Logf("✅ Node B recognizes Node A: %s (%s)", peer.Name, peer.ID)
			break
		}
	}
	assert.True(t, nodeAFoundInB, "Node B should recognize Node A by name and ID")

	// Verify Node A has connected to Node B (bidirectional connection)
	t.Log("🔍 Verifying Node A's connections...")
	nodeAPeers, err := getNodePeerInfo(ctx, nodeA)
	require.NoError(t, err, "Failed to get Node A peer info")
	assert.GreaterOrEqual(t, len(nodeAPeers), 1, "Node A should have at least 1 connected peer")

	// Get Node B's info to verify the bidirectional connection
	nodeBInfo, err := getNodeConnectionInfo(ctx, nodeB)
	require.NoError(t, err, "Failed to get Node B connection info")

	var nodeBFoundInA bool
	for _, peer := range nodeAPeers {
		if peer.ID == nodeBInfo.PeerID && peer.Name == "node-b" {
			nodeBFoundInA = true
			t.Logf("✅ Node A recognizes Node B: %s (%s)", peer.Name, peer.ID)
			break
		}
	}
	assert.True(t, nodeBFoundInA, "Node A should recognize Node B by name and ID")

	// Final verification: Both nodes should have exactly 1 peer (each other)
	t.Log("🎯 Final verification...")
	if nodeAFoundInB && nodeBFoundInA {
		t.Log("🎉 SUCCESS: Bidirectional connection established between isolated nodes!")
		t.Logf("   Node A (%s) ↔ Node B (%s)", nodeAInfo.PeerID[:12]+"...", nodeBInfo.PeerID[:12]+"...")
	}

	t.Log("✅ Two isolated nodes connection test completed successfully")
}

// getDockerfileContent returns the Dockerfile content for the test containers
func getDockerfileContent() string {
	return `
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git gcc musl-dev sqlite
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o distributed-app ./cmd/distributed-app

FROM alpine:latest
RUN apk add --no-cache ca-certificates sqlite
WORKDIR /app
COPY --from=builder /app/distributed-app .
COPY --from=builder /app/web ./web
RUN mkdir -p /app/space184
EXPOSE 6996 9000-9010
CMD ["./distributed-app"]
`
}

// startIsolatedNode starts a containerized node with the given name
func startIsolatedNode(ctx context.Context, nodeName, dockerfile string) (testcontainers.Container, error) {
	// Find the project root dynamically
	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find project root: %w", err)
	}

	// Write the Dockerfile to the project root temporarily
	dockerfilePath := filepath.Join(projectRoot, "Dockerfile.test")
	err = os.WriteFile(dockerfilePath, []byte(dockerfile), 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write temporary Dockerfile: %w", err)
	}

	// Clean up the temporary Dockerfile after container creation
	defer func() {
		os.Remove(dockerfilePath)
	}()

	req := testcontainers.ContainerRequest{
		Name: fmt.Sprintf("isolated-node-%s", nodeName),
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       projectRoot,
			Dockerfile:    "Dockerfile.test",
			PrintBuildLog: true,
		},
		ExposedPorts: []string{"6996/tcp", "9000/tcp"},
		WaitingFor:   wait.ForLog("P2P Host created successfully").WithStartupTimeout(120 * time.Second),
		Env: map[string]string{
			"NODE_NAME": nodeName,
		},
	}

	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

// getNodeConnectionInfo retrieves connection information from a node's API
func getNodeConnectionInfo(ctx context.Context, container testcontainers.Container) (*ConnectionInfoResponse, error) {
	url, err := buildAPIURL(ctx, container, "/api/connection-info")
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var info ConnectionInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &info, nil
}

// getNodePeerInfo retrieves connected peer information from a node's API
func getNodePeerInfo(ctx context.Context, container testcontainers.Container) ([]PeerInfoResponse, error) {
	url, err := buildAPIURL(ctx, container, "/api/peer-info")
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get peer info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var peers []PeerInfoResponse
	fmt.Println("---------------")
	fmt.Println(resp.Body)
	fmt.Println("---------------")
	if err := json.NewDecoder(resp.Body).Decode(&peers); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return peers, nil
}

// connectNodeToTarget connects a node to another node via API
func connectNodeToTarget(ctx context.Context, container testcontainers.Container, targetIP string, targetPort int, targetPeerID string) error {
	url, err := buildAPIURL(ctx, container, "/api/connect-ip")
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"ip":     targetIP,
		"port":   targetPort,
		"peerId": targetPeerID,
	}

	payloadBytes, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", strings.NewReader(string(payloadBytes)))
	if err != nil {
		return fmt.Errorf("failed to connect to node: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("connect API returned status %d", resp.StatusCode)
	}

	return nil
}

// buildAPIURL constructs the API URL for a container endpoint
func buildAPIURL(ctx context.Context, container testcontainers.Container, endpoint string) (string, error) {
	host, err := container.Host(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get container host: %w", err)
	}

	// Try to get the mapped port for 6996
	port, err := container.MappedPort(ctx, "6996")
	if err != nil {
		// Debug: List all exposed ports
		expose, _ := container.Ports(ctx)
		exposedPorts := make([]string, 0)
		for nat := range expose {
			exposedPorts = append(exposedPorts, string(nat))
		}
		return "", fmt.Errorf("failed to get mapped port 6996 (exposed ports: %v): %w", exposedPorts, err)
	}

	return fmt.Sprintf("http://%s:%s%s", host, port.Port(), endpoint), nil
}

// findProjectRoot finds the project root directory by looking for go.mod
func findProjectRoot() (string, error) {
	// Start from current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up the directory tree looking for go.mod
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the root directory
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("could not find go.mod file in directory tree")
}
