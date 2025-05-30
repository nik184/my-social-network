package integration

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"my-social-network/internal/services"
)

// TestNodeConfig represents configuration for a test node
type TestNodeConfig struct {
	Name     string
	TempDir  string
	Isolated bool // If true, simulate NAT/firewall isolation
}

// TestNode represents a node in the test network
type TestNode struct {
	*services.AppService
	Config  TestNodeConfig
	cleanup func()
}

// NewTestNode creates a new test node with the given configuration
func NewTestNode(t *testing.T, config TestNodeConfig) *TestNode {
	t.Helper()

	if config.TempDir == "" {
		var err error
		config.TempDir, err = os.MkdirTemp("", fmt.Sprintf("node_%s_", config.Name))
		require.NoError(t, err)
	}

	appService, cleanup := setupTestNode(t, config.TempDir, config.Name)
	
	return &TestNode{
		AppService: appService,
		Config:     config,
		cleanup:    cleanup,
	}
}

// Close cleans up the test node
func (tn *TestNode) Close() {
	if tn.cleanup != nil {
		tn.cleanup()
	}
	if tn.Config.TempDir != "" {
		os.RemoveAll(tn.Config.TempDir)
	}
}

// GetConnectableAddress returns the IP and port that other nodes can use to connect
func (tn *TestNode) GetConnectableAddress() (string, int) {
	connectionInfo := tn.P2PService.GetConnectionInfo()
	return findConnectableAddress(connectionInfo.LocalAddresses)
}

// findConnectableAddress extracts IP and port from libp2p addresses
func findConnectableAddress(addresses []string) (string, int) {
	for _, addr := range addresses {
		if ip, port := extractIPAndPort(addr); ip != "" && port != 0 && ip != "127.0.0.1" {
			return ip, port
		}
	}
	
	// Fallback to localhost if no other address found
	for _, addr := range addresses {
		if ip, port := extractIPAndPort(addr); ip == "127.0.0.1" && port != 0 {
			return ip, port
		}
	}
	
	return "", 0
}

// ConnectTo attempts to connect this node to another test node
func (tn *TestNode) ConnectTo(t *testing.T, target *TestNode) error {
	t.Helper()
	
	ip, port := target.GetConnectableAddress()
	if ip == "" || port == 0 {
		return fmt.Errorf("could not find connectable address for target node %s", target.Config.Name)
	}
	
	targetConnectionInfo := target.P2PService.GetConnectionInfo()
	_, err := tn.P2PService.ConnectByIP(ip, port, targetConnectionInfo.PeerID)
	return err
}

// WaitForConnection waits for this node to establish a connection with the target
func (tn *TestNode) WaitForConnection(t *testing.T, targetPeerID string, timeout time.Duration) bool {
	t.Helper()
	
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		peers := tn.P2PService.GetConnectedPeers()
		for _, peer := range peers {
			if peer.String() == targetPeerID {
				return true
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

// GetPeerByName returns peer info for a peer with the given name
func (tn *TestNode) GetPeerByName(name string) *services.PeerInfo {
	peerInfo := tn.P2PService.GetConnectedPeerInfo()
	for _, info := range peerInfo {
		if info.Name == name {
			return info
		}
	}
	return nil
}

// HasConnectionRecord checks if this node has a connection record for the given peer name
func (tn *TestNode) HasConnectionRecord(t *testing.T, peerName string) bool {
	t.Helper()
	
	history, err := tn.DatabaseService.GetConnectionHistory()
	require.NoError(t, err)
	
	for _, record := range history {
		if record.PeerName == peerName {
			return true
		}
	}
	return false
}

// TestNetwork represents a collection of test nodes for integration testing
type TestNetwork struct {
	nodes   map[string]*TestNode
	cleanup []func()
}

// NewTestNetwork creates a new test network
func NewTestNetwork() *TestNetwork {
	return &TestNetwork{
		nodes: make(map[string]*TestNode),
	}
}

// AddNode adds a node to the test network
func (tn *TestNetwork) AddNode(t *testing.T, config TestNodeConfig) *TestNode {
	t.Helper()
	
	node := NewTestNode(t, config)
	tn.nodes[config.Name] = node
	tn.cleanup = append(tn.cleanup, node.Close)
	
	return node
}

// GetNode returns a node by name
func (tn *TestNetwork) GetNode(name string) *TestNode {
	return tn.nodes[name]
}

// ConnectNodes connects two nodes in the network
func (tn *TestNetwork) ConnectNodes(t *testing.T, from, to string) error {
	t.Helper()
	
	fromNode := tn.nodes[from]
	toNode := tn.nodes[to]
	
	if fromNode == nil {
		return fmt.Errorf("node %s not found", from)
	}
	if toNode == nil {
		return fmt.Errorf("node %s not found", to)
	}
	
	return fromNode.ConnectTo(t, toNode)
}

// WaitForStabilization waits for all connections in the network to stabilize
func (tn *TestNetwork) WaitForStabilization(t *testing.T, duration time.Duration) {
	t.Helper()
	t.Logf("Waiting %v for network stabilization...", duration)
	time.Sleep(duration)
}

// Close cleans up the entire test network
func (tn *TestNetwork) Close() {
	for i := len(tn.cleanup) - 1; i >= 0; i-- {
		tn.cleanup[i]()
	}
}

// VerifyNetworkTopology verifies that the network has the expected topology
func (tn *TestNetwork) VerifyNetworkTopology(t *testing.T, expectedConnections map[string][]string) {
	t.Helper()
	
	for nodeName, expectedPeers := range expectedConnections {
		node := tn.nodes[nodeName]
		require.NotNil(t, node, "Node %s should exist", nodeName)
		
		actualPeers := node.P2PService.GetConnectedPeers()
		require.Len(t, actualPeers, len(expectedPeers), 
			"Node %s should have %d connected peers", nodeName, len(expectedPeers))
		
		// Verify each expected peer is connected
		peerInfo := node.P2PService.GetConnectedPeerInfo()
		for _, expectedPeerName := range expectedPeers {
			found := false
			for _, info := range peerInfo {
				if info.Name == expectedPeerName {
					found = true
					break
				}
			}
			require.True(t, found, 
				"Node %s should be connected to %s", nodeName, expectedPeerName)
		}
	}
}

// CreateStarTopology creates a star network topology with one central relay
func CreateStarTopology(t *testing.T, relayName string, clientNames []string) *TestNetwork {
	t.Helper()
	
	network := NewTestNetwork()
	
	// Add relay node
	relay := network.AddNode(t, TestNodeConfig{Name: relayName})
	
	// Add client nodes and connect them to relay
	for _, clientName := range clientNames {
		client := network.AddNode(t, TestNodeConfig{Name: clientName})
		
		// Connect client to relay
		err := client.ConnectTo(t, relay)
		require.NoError(t, err, "Client %s should connect to relay %s", clientName, relayName)
	}
	
	// Wait for connections to stabilize
	network.WaitForStabilization(t, 5*time.Second)
	
	return network
}

// CreateMeshTopology creates a full mesh network topology
func CreateMeshTopology(t *testing.T, nodeNames []string) *TestNetwork {
	t.Helper()
	
	network := NewTestNetwork()
	
	// Add all nodes
	nodes := make([]*TestNode, len(nodeNames))
	for i, name := range nodeNames {
		nodes[i] = network.AddNode(t, TestNodeConfig{Name: name})
	}
	
	// Connect each node to every other node
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			err := nodes[i].ConnectTo(t, nodes[j])
			require.NoError(t, err, "Node %s should connect to %s", 
				nodeNames[i], nodeNames[j])
		}
	}
	
	// Wait for connections to stabilize
	network.WaitForStabilization(t, 5*time.Second)
	
	return network
}

// LogNetworkStatus logs the current status of all nodes in the network
func (tn *TestNetwork) LogNetworkStatus(t *testing.T) {
	t.Helper()
	
	t.Log("=== Network Status ===")
	for name, node := range tn.nodes {
		peers := node.P2PService.GetConnectedPeers()
		peerInfo := node.P2PService.GetConnectedPeerInfo()
		
		peerNames := make([]string, 0, len(peerInfo))
		for _, info := range peerInfo {
			peerNames = append(peerNames, info.Name)
		}
		
		t.Logf("Node %s: %d peers %v", name, len(peers), peerNames)
	}
	t.Log("=====================")
}

// WaitForPeerDiscovery waits for a node to discover a specific second-degree peer
func (tn *TestNode) WaitForPeerDiscovery(t *testing.T, targetPeerName string, timeout time.Duration) bool {
	t.Helper()
	
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		secondDegree, err := tn.P2PService.GetSecondDegreeConnections()
		if err == nil {
			for _, peer := range secondDegree.Peers {
				if peer.PeerName == targetPeerName {
					return true
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}