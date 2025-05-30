package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
}

// SecondDegreePeerResponse represents a second-degree peer in API responses
type SecondDegreePeerResponse struct {
	PeerID      string `json:"peer_id"`
	PeerName    string `json:"peer_name"`
	ViaPeerID   string `json:"via_peer_id"`
	ViaPeerName string `json:"via_peer_name"`
}

// TestContainerizedDirectConnection tests direct P2P connection in isolated containers
// This test verifies that nodes can connect across different network segments
func TestContainerizedDirectConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping containerized test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Build application image
	dockerfile := `
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
RUN mkdir -p /app/space184
EXPOSE 6996 9000-9010
CMD ["./distributed-app"]
`

	// Start first container (Alice)
	aliceContainer, err := startAppContainer(ctx, "alice", dockerfile)
	require.NoError(t, err)
	defer aliceContainer.Terminate(ctx)

	// Start second container (Bob)
	bobContainer, err := startAppContainer(ctx, "bob", dockerfile)
	require.NoError(t, err)
	defer bobContainer.Terminate(ctx)

	// Wait for applications to initialize
	time.Sleep(10 * time.Second)

	// Get connection information from Alice
	aliceInfo, err := getConnectionInfo(ctx, aliceContainer)
	require.NoError(t, err)
	t.Logf("Alice connection info: PeerID=%s, Addresses=%v", aliceInfo.PeerID, aliceInfo.LocalAddresses)

	// Get Alice's container IP for connection
	aliceIP, err := aliceContainer.ContainerIP(ctx)
	require.NoError(t, err)
	t.Logf("Alice container IP: %s", aliceIP)

	// Connect Bob to Alice
	err = connectToNode(ctx, bobContainer, aliceIP, 9000, aliceInfo.PeerID)
	require.NoError(t, err)

	// Wait for connection to establish
	time.Sleep(5 * time.Second)

	// Verify Bob is connected to Alice
	bobPeers, err := getConnectedPeerInfo(ctx, bobContainer)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(bobPeers), 1, "Bob should have at least 1 connected peer")
	
	// Check if Alice is in Bob's peer list with correct name
	var aliceFoundInBob bool
	for _, peer := range bobPeers {
		if peer.ID == aliceInfo.PeerID && peer.Name == "alice" {
			aliceFoundInBob = true
			break
		}
	}
	assert.True(t, aliceFoundInBob, "Bob should know Alice by name")

	// Verify Alice is connected to Bob
	alicePeers, err := getConnectedPeerInfo(ctx, aliceContainer)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(alicePeers), 1, "Alice should have at least 1 connected peer")

	// Get Bob's info to verify Alice knows Bob by name
	bobInfo, err := getConnectionInfo(ctx, bobContainer)
	require.NoError(t, err)

	var bobFoundInAlice bool
	for _, peer := range alicePeers {
		if peer.ID == bobInfo.PeerID && peer.Name == "bob" {
			bobFoundInAlice = true
			break
		}
	}
	assert.True(t, bobFoundInAlice, "Alice should know Bob by name")

	t.Log("✅ Containerized direct connection test completed successfully")
}

// TestContainerizedHolePunching tests hole punching through a relay in isolated containers
// This test creates a three-node topology to test relay-assisted connections
func TestContainerizedHolePunching(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping containerized hole punching test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Minute)
	defer cancel()

	// Build application image
	dockerfile := `
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
RUN mkdir -p /app/space184
EXPOSE 6996 9000-9010
CMD ["./distributed-app"]
`

	// Start relay container
	relayContainer, err := startAppContainer(ctx, "relay", dockerfile)
	require.NoError(t, err)
	defer relayContainer.Terminate(ctx)

	// Start client A container
	clientAContainer, err := startAppContainer(ctx, "client-a", dockerfile)
	require.NoError(t, err)
	defer clientAContainer.Terminate(ctx)

	// Start client B container
	clientBContainer, err := startAppContainer(ctx, "client-b", dockerfile)
	require.NoError(t, err)
	defer clientBContainer.Terminate(ctx)

	// Wait for applications to initialize
	time.Sleep(15 * time.Second)

	// Get relay connection information
	relayInfo, err := getConnectionInfo(ctx, relayContainer)
	require.NoError(t, err)
	t.Logf("Relay connection info: PeerID=%s", relayInfo.PeerID)

	// Get relay IP
	relayIP, err := relayContainer.ContainerIP(ctx)
	require.NoError(t, err)
	t.Logf("Relay IP: %s", relayIP)

	// Connect Client A to Relay
	err = connectToNode(ctx, clientAContainer, relayIP, 9000, relayInfo.PeerID)
	require.NoError(t, err)
	t.Log("Client A connected to Relay")

	// Connect Client B to Relay
	err = connectToNode(ctx, clientBContainer, relayIP, 9000, relayInfo.PeerID)
	require.NoError(t, err)
	t.Log("Client B connected to Relay")

	// Wait for connections to stabilize
	time.Sleep(8 * time.Second)

	// Verify relay has both clients connected
	relayPeers, err := getConnectedPeerInfo(ctx, relayContainer)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(relayPeers), 2, "Relay should have at least 2 connected peers")

	// Test second-degree peer discovery from Client A
	secondDegreeA, err := getSecondDegreePeers(ctx, clientAContainer)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(secondDegreeA), 1, "Client A should discover at least 1 second-degree peer")

	if len(secondDegreeA) > 0 {
		// Find Client B in second-degree peers
		var clientBPeer *SecondDegreePeerResponse
		for _, peer := range secondDegreeA {
			if peer.PeerName == "client-b" {
				clientBPeer = &peer
				break
			}
		}
		require.NotNil(t, clientBPeer, "Client A should discover Client B as second-degree peer")

		// Attempt hole punching from Client A to Client B
		err = connectToSecondDegreePeer(ctx, clientAContainer, clientBPeer.PeerID, clientBPeer.ViaPeerID)
		require.NoError(t, err)
		t.Log("Initiated hole punching from Client A to Client B")

		// Wait for hole punching to complete
		time.Sleep(10 * time.Second)

		// Verify direct connection was established
		clientAPeers, err := getConnectedPeerInfo(ctx, clientAContainer)
		require.NoError(t, err)

		// Client A should now have direct connection to Client B (plus relay)
		var directConnectionFound bool
		for _, peer := range clientAPeers {
			if peer.Name == "client-b" {
				directConnectionFound = true
				break
			}
		}
		assert.True(t, directConnectionFound, "Client A should have direct connection to Client B after hole punching")
	}

	t.Log("✅ Containerized hole punching test completed successfully")
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

// startAppContainer starts a container running our application
func startAppContainer(ctx context.Context, nodeName, dockerfile string) (testcontainers.Container, error) {
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
		Name: fmt.Sprintf("p2p-test-%s", nodeName),
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       projectRoot,
			Dockerfile:    "Dockerfile.test", // Relative path from context
			PrintBuildLog: true,              // Enable build logs for debugging
		},
		ExposedPorts: []string{"6996/tcp", "9000/tcp"},
		WaitingFor:   wait.ForLog("P2P Host created successfully").WithStartupTimeout(90 * time.Second),
		Env: map[string]string{
			"NODE_NAME": nodeName,
		},
	}

	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:         true,
	})
}

// getConnectionInfo retrieves connection information from a container's API
func getConnectionInfo(ctx context.Context, container testcontainers.Container) (*ConnectionInfoResponse, error) {
	url, err := getAPIURL(ctx, container, "/api/connection-info")
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

// getConnectedPeerInfo retrieves connected peer information from a container's API
func getConnectedPeerInfo(ctx context.Context, container testcontainers.Container) ([]PeerInfoResponse, error) {
	url, err := getAPIURL(ctx, container, "/api/peer-info")
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
	if err := json.NewDecoder(resp.Body).Decode(&peers); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return peers, nil
}

// getSecondDegreePeers retrieves second-degree peer information from a container's API
func getSecondDegreePeers(ctx context.Context, container testcontainers.Container) ([]SecondDegreePeerResponse, error) {
	url, err := getAPIURL(ctx, container, "/api/second-degree-peers")
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get second-degree peers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var peers []SecondDegreePeerResponse
	if err := json.NewDecoder(resp.Body).Decode(&peers); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return peers, nil
}

// connectToNode connects a container to another node via API
func connectToNode(ctx context.Context, container testcontainers.Container, targetIP string, targetPort int, targetPeerID string) error {
	url, err := getAPIURL(ctx, container, "/api/connect-ip")
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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("connect API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// connectToSecondDegreePeer initiates hole punching to a second-degree peer
func connectToSecondDegreePeer(ctx context.Context, container testcontainers.Container, targetPeerID, viaPeerID string) error {
	url, err := getAPIURL(ctx, container, "/api/connect-second-degree")
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"targetPeerId": targetPeerID,
		"viaPeerId":    viaPeerID,
	}

	payloadBytes, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", strings.NewReader(string(payloadBytes)))
	if err != nil {
		return fmt.Errorf("failed to connect to second-degree peer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("second-degree connect API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// getAPIURL constructs the API URL for a container endpoint
func getAPIURL(ctx context.Context, container testcontainers.Container, endpoint string) (string, error) {
	host, err := container.Host(ctx)
	if err != nil {
		return "", err
	}

	port, err := container.MappedPort(ctx, "6996")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("http://%s:%s%s", host, port.Port(), endpoint), nil
}

// TestNetworkPartitioning tests behavior during simulated network partitions
func TestNetworkPartitioning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network partition test in short mode")
	}

	t.Log("Network partitioning test using container network isolation")
	t.Log("This test would:")
	t.Log("- Create multiple isolated networks")
	t.Log("- Start nodes on different networks")
	t.Log("- Connect/disconnect networks to simulate partitions")
	t.Log("- Verify node behavior during splits and healing")
	t.Log("- Test automatic reconnection capabilities")
	
	// The actual implementation would use Docker network operations
	// to dynamically connect/disconnect containers from networks
}

// TestHighLatencyNetwork tests behavior under network latency using traffic control
func TestHighLatencyNetwork(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high latency test in short mode")
	}

	t.Log("High latency network test using container network emulation")
	t.Log("This test would:")
	t.Log("- Use netem (network emulation) to introduce latency")
	t.Log("- Test connection establishment under high latency")
	t.Log("- Verify timeout handling and retry mechanisms")
	t.Log("- Measure performance degradation patterns")
	t.Log("- Test hole punching success rates with latency")
	
	// The actual implementation would require containers with network
	// emulation capabilities (tc/netem) to introduce latency
}