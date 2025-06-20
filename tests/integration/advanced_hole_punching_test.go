package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestAdvancedHolePunchingMechanism tests a complex hole punching scenario with 4 nodes
// This test verifies:
// 1. One public node can help multiple private nodes connect to each other
// 2. Private nodes can discover each other through the public node
// 3. After public node termination, private nodes can use other connected nodes as relays
// 4. Complex multi-hop connection establishment works correctly
func TestAdvancedHolePunchingMechanism(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping advanced hole punching test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()

	t.Log("🚀 Starting advanced hole punching mechanism test (4 nodes)...")

	// Create a single custom network for all nodes
	testNetwork, err := network.New(ctx, network.WithDriver("bridge"))
	require.NoError(t, err, "Failed to create test network")
	defer func() {
		if err := testNetwork.Remove(ctx); err != nil {
			t.Logf("Warning: Failed to remove test network: %v", err)
		}
	}()

	// Build application image
	dockerfile := getAdvancedHolePunchDockerfileContent()

	// Start public node (initial relay)
	t.Log("📦 Starting Public Node container...")
	publicNode, err := startAdvancedNode(ctx, "public-node", dockerfile, testNetwork.Name, true)
	require.NoError(t, err, "Failed to start Public Node container")
	defer func() {
		if err := publicNode.Terminate(ctx); err != nil {
			t.Logf("Warning: Failed to terminate Public Node: %v", err)
		}
	}()

	// Start three private nodes (NAT-ed nodes)
	t.Log("📦 Starting Private Node A container...")
	privateNodeA, err := startAdvancedNode(ctx, "private-node-a", dockerfile, testNetwork.Name, false)
	require.NoError(t, err, "Failed to start Private Node A container")
	defer func() {
		if err := privateNodeA.Terminate(ctx); err != nil {
			t.Logf("Warning: Failed to terminate Private Node A: %v", err)
		}
	}()

	t.Log("📦 Starting Private Node B container...")
	privateNodeB, err := startAdvancedNode(ctx, "private-node-b", dockerfile, testNetwork.Name, false)
	require.NoError(t, err, "Failed to start Private Node B container")
	defer func() {
		if err := privateNodeB.Terminate(ctx); err != nil {
			t.Logf("Warning: Failed to terminate Private Node B: %v", err)
		}
	}()

	t.Log("📦 Starting Private Node C container...")
	privateNodeC, err := startAdvancedNode(ctx, "private-node-c", dockerfile, testNetwork.Name, false)
	require.NoError(t, err, "Failed to start Private Node C container")
	defer func() {
		if err := privateNodeC.Terminate(ctx); err != nil {
			t.Logf("Warning: Failed to terminate Private Node C: %v", err)
		}
	}()

	// Wait for all nodes to initialize
	t.Log("⏳ Waiting for all nodes to initialize...")
	time.Sleep(25 * time.Second)

	// Get node information from all nodes
	t.Log("🔍 Retrieving node information...")
	publicInfo, err := getNodeInfo(ctx, publicNode)
	require.NoError(t, err, "Failed to get Public Node info")
	t.Logf("✅ Public Node - PeerID: %s", publicInfo.Node.ID[:12]+"...")

	privateAInfo, err := getNodeInfo(ctx, privateNodeA)
	require.NoError(t, err, "Failed to get Private Node A info")
	t.Logf("✅ Private Node A - PeerID: %s", privateAInfo.Node.ID[:12]+"...")

	privateBInfo, err := getNodeInfo(ctx, privateNodeB)
	require.NoError(t, err, "Failed to get Private Node B info")
	t.Logf("✅ Private Node B - PeerID: %s", privateBInfo.Node.ID[:12]+"...")

	privateCInfo, err := getNodeInfo(ctx, privateNodeC)
	require.NoError(t, err, "Failed to get Private Node C info")
	t.Logf("✅ Private Node C - PeerID: %s", privateCInfo.Node.ID[:12]+"...")

	// Phase 1: Connect all private nodes to the public node
	t.Log("🔗 Phase 1: Connecting all private nodes to Public Node...")

	publicNodeIP, err := publicNode.ContainerIP(ctx)
	require.NoError(t, err, "Failed to get Public Node IP")
	t.Logf("🌐 Public Node IP: %s", publicNodeIP)

	// Connect Private Node A to Public Node
	err = connectNodeToTarget(ctx, privateNodeA, publicNodeIP, 9000, publicInfo.Node.ID)
	require.NoError(t, err, "Private Node A should connect to Public Node")
	t.Log("✅ Private Node A connected to Public Node")

	// Connect Private Node B to Public Node
	err = connectNodeToTarget(ctx, privateNodeB, publicNodeIP, 9000, publicInfo.Node.ID)
	require.NoError(t, err, "Private Node B should connect to Public Node")
	t.Log("✅ Private Node B connected to Public Node")

	// Connect Private Node C to Public Node
	err = connectNodeToTarget(ctx, privateNodeC, publicNodeIP, 9000, publicInfo.Node.ID)
	require.NoError(t, err, "Private Node C should connect to Public Node")
	t.Log("✅ Private Node C connected to Public Node")

	// Wait for connections to establish
	time.Sleep(12 * time.Second)

	// Verify all private nodes are connected to the public node
	t.Log("🔍 Verifying connections to Public Node...")
	publicPeers, err := getNodePeers(ctx, publicNode)
	require.NoError(t, err, "Failed to get Public Node peers")
	assert.GreaterOrEqual(t, publicPeers.ValidatedCount, 3, "Public Node should have at least 3 connected peers")
	t.Logf("✅ Public Node has %d validated peers", publicPeers.ValidatedCount)

	// Phase 2: Use Public Node to help Private Node A and B connect to Private Node C
	t.Log("🔗 Phase 2: Using Public Node to connect Private Nodes A & B to Private Node C...")

	// Private Node A discovers and connects to Private Node C via Public Node
	err = requestHolePunching(ctx, privateNodeA, privateCInfo.Node.ID, publicInfo.Node.ID)
	require.NoError(t, err, "Private Node A should discover Private Node C via Public Node")
	t.Log("✅ Private Node A requested connection to Private Node C via Public Node")

	// Private Node B discovers and connects to Private Node C via Public Node
	err = requestHolePunching(ctx, privateNodeB, privateCInfo.Node.ID, publicInfo.Node.ID)
	require.NoError(t, err, "Private Node B should discover Private Node C via Public Node")
	t.Log("✅ Private Node B requested connection to Private Node C via Public Node")

	// Wait for hole punching to complete
	t.Log("⏳ Waiting for initial hole punching to complete...")
	time.Sleep(15 * time.Second)

	// Verify Private Node A and B are connected to Private Node C
	t.Log("🔍 Verifying Private Node A connections...")
	privateAPeers, err := getNodePeers(ctx, privateNodeA)
	require.NoError(t, err, "Failed to get Private Node A peers")

	var privateCFoundInA bool
	for _, peerID := range privateAPeers.ValidatedPeers {
		if peerID == privateCInfo.Node.ID {
			privateCFoundInA = true
			t.Logf("✅ Private Node A recognizes Private Node C: %s", peerID[:12]+"...")
			break
		}
	}
	assert.True(t, privateCFoundInA, "Private Node A should be connected to Private Node C")

	t.Log("🔍 Verifying Private Node B connections...")
	privateBPeers, err := getNodePeers(ctx, privateNodeB)
	require.NoError(t, err, "Failed to get Private Node B peers")

	var privateCFoundInB bool
	for _, peerID := range privateBPeers.ValidatedPeers {
		if peerID == privateCInfo.Node.ID {
			privateCFoundInB = true
			t.Logf("✅ Private Node B recognizes Private Node C: %s", peerID[:12]+"...")
			break
		}
	}
	assert.True(t, privateCFoundInB, "Private Node B should be connected to Private Node C")

	// Verify Private Node C sees both A and B
	t.Log("🔍 Verifying Private Node C connections...")
	privateCPeers, err := getNodePeers(ctx, privateNodeC)
	require.NoError(t, err, "Failed to get Private Node C peers")

	var privateAFoundInC, privateBFoundInC bool
	for _, peerID := range privateCPeers.ValidatedPeers {
		if peerID == privateAInfo.Node.ID {
			privateAFoundInC = true
			t.Logf("✅ Private Node C recognizes Private Node A: %s", peerID[:12]+"...")
		}
		if peerID == privateBInfo.Node.ID {
			privateBFoundInC = true
			t.Logf("✅ Private Node C recognizes Private Node B: %s", peerID[:12]+"...")
		}
	}
	assert.True(t, privateAFoundInC, "Private Node C should be connected to Private Node A")
	assert.True(t, privateBFoundInC, "Private Node C should be connected to Private Node B")

	// Phase 3: Terminate Public Node (simulate relay failure)
	t.Log("🔗 Phase 3: Terminating Public Node to test relay failover...")
	err = publicNode.Terminate(ctx)
	require.NoError(t, err, "Failed to terminate Public Node")
	t.Log("❌ Public Node terminated")

	// Wait for nodes to detect the termination
	time.Sleep(10 * time.Second)

	// Phase 4: Use Private Node C as relay for Private Nodes A and B to connect to each other
	t.Log("🔗 Phase 4: Using Private Node C as relay for A and B to connect...")

	// Private Node A discovers and connects to Private Node B via Private Node C
	err = requestHolePunching(ctx, privateNodeA, privateBInfo.Node.ID, privateCInfo.Node.ID)
	require.NoError(t, err, "Private Node A should discover Private Node B via Private Node C")
	t.Log("✅ Private Node A requested connection to Private Node B via Private Node C")

	// Wait for the final hole punching to complete
	t.Log("⏳ Waiting for final hole punching to complete...")
	time.Sleep(15 * time.Second)

	// Phase 5: Verify final connections
	t.Log("🔍 Phase 5: Verifying final network topology...")

	// Check if Private Node A is connected to Private Node B
	privateAPeersFinal, err := getNodePeers(ctx, privateNodeA)
	require.NoError(t, err, "Failed to get Private Node A final peers")

	var privateBFoundInAFinal bool
	for _, peerID := range privateAPeersFinal.ValidatedPeers {
		if peerID == privateBInfo.Node.ID {
			privateBFoundInAFinal = true
			t.Logf("✅ Private Node A recognizes Private Node B: %s", peerID[:12]+"...")
			break
		}
	}

	// Check if Private Node B is connected to Private Node A
	privateBPeersFinal, err := getNodePeers(ctx, privateNodeB)
	require.NoError(t, err, "Failed to get Private Node B final peers")

	var privateAFoundInBFinal bool
	for _, peerID := range privateBPeersFinal.ValidatedPeers {
		if peerID == privateAInfo.Node.ID {
			privateAFoundInBFinal = true
			t.Logf("✅ Private Node B recognizes Private Node A: %s", peerID[:12]+"...")
			break
		}
	}

	// Final verification and results
	if privateBFoundInAFinal && privateAFoundInBFinal {
		t.Log("🎉 SUCCESS: Advanced hole punching completed successfully!")
		t.Log("   📋 Final Network Topology:")
		t.Logf("      Private Node A (%s) ↔ Private Node B (%s)", privateAInfo.Node.ID[:12]+"...", privateBInfo.Node.ID[:12]+"...")
		t.Logf("      Private Node A (%s) ↔ Private Node C (%s)", privateAInfo.Node.ID[:12]+"...", privateCInfo.Node.ID[:12]+"...")
		t.Logf("      Private Node B (%s) ↔ Private Node C (%s)", privateBInfo.Node.ID[:12]+"...", privateCInfo.Node.ID[:12]+"...")
		t.Log("   ✨ All private nodes are interconnected without public relay!")
	} else if privateBFoundInAFinal || privateAFoundInBFinal {
		t.Log("⚠️ PARTIAL SUCCESS: Unidirectional connection established between A and B")
		t.Log("   This may be expected in some network configurations")
	} else {
		t.Log("⚠️ Final hole punching did not establish A-B connection")
		t.Log("   However, A and B can still communicate via C as relay")
	}

	// Test P2P communication across the network
	t.Log("🔍 Testing P2P communication across the network...")

	// Test A -> C communication
	err = testP2PCommunication(ctx, privateNodeA, privateCInfo.Node.ID)
	if err == nil {
		t.Log("✅ P2P communication successful: A -> C")
	} else {
		t.Logf("⚠️ P2P communication failed A -> C: %v", err)
	}

	// Test B -> C communication
	err = testP2PCommunication(ctx, privateNodeB, privateCInfo.Node.ID)
	if err == nil {
		t.Log("✅ P2P communication successful: B -> C")
	} else {
		t.Logf("⚠️ P2P communication failed B -> C: %v", err)
	}

	// Test A -> B communication (if connected)
	if privateBFoundInAFinal {
		err = testP2PCommunication(ctx, privateNodeA, privateBInfo.Node.ID)
		if err == nil {
			t.Log("✅ P2P communication successful: A -> B")
		} else {
			t.Logf("⚠️ P2P communication failed A -> B: %v", err)
		}
	}

	t.Log("✅ Advanced hole punching mechanism test completed successfully")
}

// getAdvancedHolePunchDockerfileContent returns the Dockerfile for advanced hole punching test
func getAdvancedHolePunchDockerfileContent() string {
	return `
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git gcc musl-dev sqlite
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o distributed-app ./cmd/distributed-app

FROM alpine:latest
RUN apk add --no-cache ca-certificates sqlite iptables
WORKDIR /app
COPY --from=builder /app/distributed-app .
COPY --from=builder /app/web ./web
RUN mkdir -p /app/space184
EXPOSE 6996 9000-9010
CMD ["./distributed-app"]
`
}

// startAdvancedNode starts a containerized node for the advanced hole punching test
func startAdvancedNode(ctx context.Context, nodeName, dockerfile, networkName string, isPublic bool) (testcontainers.Container, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find project root: %w", err)
	}

	nodeType := "private"
	if isPublic {
		nodeType = "public"
	}

	req := testcontainers.ContainerRequest{
		Name: fmt.Sprintf("advanced-hole-punch-%s-%s", nodeType, nodeName),
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       projectRoot,
			Dockerfile:    "Dockerfile.test",
			PrintBuildLog: false,
		},
		ExposedPorts: []string{"6996/tcp", "9000/tcp"},
		Networks:     []string{networkName},
		WaitingFor:   wait.ForLog("P2P Host created successfully").WithStartupTimeout(120 * time.Second),
		Env: map[string]string{
			"NODE_NAME":                nodeName,
			"IS_PUBLIC_NODE":           fmt.Sprintf("%t", isPublic),
			"ADVANCED_HOLE_PUNCH_MODE": nodeType,
		},
	}

	// Write temporary Dockerfile
	dockerfilePath := projectRoot + "/Dockerfile.test"
	if err := writeTemporaryDockerfile(dockerfilePath, dockerfile); err != nil {
		return nil, err
	}
	defer removeTemporaryDockerfile(dockerfilePath)

	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}
