package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"my-social-network/internal/models"
	"my-social-network/internal/services"
)

// TestDirectConnection tests direct peer-to-peer connection between two nodes
// This test verifies:
// 1. Two nodes can connect directly
// 2. They exchange names during identification
// 3. Both nodes store each other's connection information in their databases
// 4. Connection validation works properly
func TestDirectConnection(t *testing.T) {
	_ = context.Background() // Context for future use

	// Create temporary directories for each node
	tempDir1, err := os.MkdirTemp("", "node1_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir1)

	tempDir2, err := os.MkdirTemp("", "node2_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir2)

	// Setup node 1 (Alice)
	node1, cleanup1 := setupTestNode(t, tempDir1, "alice")
	defer cleanup1()

	// Setup node 2 (Bob)
	node2, cleanup2 := setupTestNode(t, tempDir2, "bob")
	defer cleanup2()

	// Give nodes time to start up
	time.Sleep(2 * time.Second)

	// Get connection info from node1
	connectionInfo := node1.P2PService.GetConnectionInfo()
	require.NotEmpty(t, connectionInfo.LocalAddresses, "Node1 should have local addresses")

	// Find a usable address and port
	var connectIP string
	var connectPort int
	for _, addr := range connectionInfo.LocalAddresses {
		if ip, port := extractIPAndPort(addr); ip != "" && port != 0 && ip != "127.0.0.1" {
			connectIP = ip
			connectPort = port
			break
		}
	}

	// Fallback to localhost if no other address found
	if connectIP == "" {
		for _, addr := range connectionInfo.LocalAddresses {
			if ip, port := extractIPAndPort(addr); ip == "127.0.0.1" && port != 0 {
				connectIP = ip
				connectPort = port
				break
			}
		}
	}

	require.NotEmpty(t, connectIP, "Should find a connectable IP address")
	require.NotZero(t, connectPort, "Should find a connectable port")

	t.Logf("Attempting to connect to node1 at %s:%d with peer ID %s", connectIP, connectPort, connectionInfo.PeerID)

	// Connect node2 to node1
	nodeInfo, err := node2.P2PService.ConnectByIP(connectIP, connectPort, connectionInfo.PeerID)
	require.NoError(t, err, "Node2 should be able to connect to Node1")
	require.NotNil(t, nodeInfo, "Should receive node info from connection")

	// Wait for connection to stabilize and validation to complete
	time.Sleep(5 * time.Second)

	// Verify connection from node2's perspective (may have multiple due to mDNS discovery)
	connectedPeers2 := node2.P2PService.GetConnectedPeers()
	assert.GreaterOrEqual(t, len(connectedPeers2), 1, "Node2 should have at least 1 connected peer")

	// Verify connection from node1's perspective (may have multiple due to mDNS discovery)
	connectedPeers1 := node1.P2PService.GetConnectedPeers()
	assert.GreaterOrEqual(t, len(connectedPeers1), 1, "Node1 should have at least 1 connected peer")

	// Check that nodes have each other's peer info with names
	peerInfo1 := node1.P2PService.GetConnectedPeerInfo()
	assert.GreaterOrEqual(t, len(peerInfo1), 1, "Node1 should have peer info for at least 1 peer")

	peerInfo2 := node2.P2PService.GetConnectedPeerInfo()
	assert.GreaterOrEqual(t, len(peerInfo2), 1, "Node2 should have peer info for at least 1 peer")

	// Verify that names were exchanged - look for specific peer by ID
	node2PeerID := node2.P2PService.GetConnectionInfo().PeerID
	node1PeerID := node1.P2PService.GetConnectionInfo().PeerID
	
	var bobInfoFromAlice *services.PeerInfo
	for _, info := range peerInfo1 {
		if info.ID.String() == node2PeerID {
			bobInfoFromAlice = info
			break
		}
	}
	require.NotNil(t, bobInfoFromAlice, "Alice should have Bob's peer info")
	assert.Equal(t, "bob", bobInfoFromAlice.Name, "Alice should know Bob's name")

	var aliceInfoFromBob *services.PeerInfo
	for _, info := range peerInfo2 {
		if info.ID.String() == node1PeerID {
			aliceInfoFromBob = info
			break
		}
	}
	require.NotNil(t, aliceInfoFromBob, "Bob should have Alice's peer info")
	assert.Equal(t, "alice", aliceInfoFromBob.Name, "Bob should know Alice's name")

	// Verify connection history is stored in databases (may have multiple connections due to mDNS)
	history1, err := node1.DatabaseService.GetConnectionHistory()
	require.NoError(t, err, "Should be able to get connection history from node1")
	assert.GreaterOrEqual(t, len(history1), 1, "Node1 should have at least 1 connection record")
	
	// Look for Bob's connection in the history
	var bobConnectionFound bool
	for _, conn := range history1 {
		if conn.PeerID == node2PeerID && conn.PeerName == "bob" {
			bobConnectionFound = true
			break
		}
	}
	assert.True(t, bobConnectionFound, "Node1 should have Bob's connection in history")

	history2, err := node2.DatabaseService.GetConnectionHistory()
	require.NoError(t, err, "Should be able to get connection history from node2")
	assert.GreaterOrEqual(t, len(history2), 1, "Node2 should have at least 1 connection record")
	
	// Look for Alice's connection in the history
	var aliceConnectionFound bool
	for _, conn := range history2 {
		if conn.PeerID == node1PeerID && conn.PeerName == "alice" {
			aliceConnectionFound = true
			break
		}
	}
	assert.True(t, aliceConnectionFound, "Node2 should have Alice's connection in history")

	// Test second-degree connections (should be empty since only 2 nodes)
	secondDegree1, err := node1.P2PService.GetSecondDegreeConnections()
	require.NoError(t, err, "Should be able to get second-degree connections from node1")
	assert.Empty(t, secondDegree1.Peers, "Node1 should have no second-degree connections with only 2 nodes")

	secondDegree2, err := node2.P2PService.GetSecondDegreeConnections()
	require.NoError(t, err, "Should be able to get second-degree connections from node2")
	assert.Empty(t, secondDegree2.Peers, "Node2 should have no second-degree connections with only 2 nodes")

	t.Log("✅ Direct connection test completed successfully")
}

// setupTestNode creates and configures a test node
func setupTestNode(t *testing.T, tempDir, nodeName string) (*services.AppService, func()) {
	t.Helper()

	// Create space184 directory
	space184Dir := filepath.Join(tempDir, "space184")
	err := os.MkdirAll(space184Dir, 0755)
	require.NoError(t, err)

	// Create a test file
	testFile := filepath.Join(space184Dir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Create a test directory service wrapper
	testDirService := &testDirectoryService{baseDir: tempDir}

	// Create database in space184 directory
	dbPath := filepath.Join(space184Dir, "node.db")
	dbService, err := services.NewDatabaseService(dbPath)
	require.NoError(t, err)

	// Set node name
	err = dbService.SetSetting("name", nodeName)
	require.NoError(t, err)

	// Create app service manually to control the directory
	appService := &services.AppService{
		DirectoryService: testDirService,
		DatabaseService:  dbService,
	}

	// Initialize P2P service
	p2pService, err := services.NewP2PService(appService, dbService)
	require.NoError(t, err)
	appService.P2PService = p2pService

	// Initialize monitor service
	monitorService, err := services.NewMonitorService(testDirService, appService)
	require.NoError(t, err)
	appService.MonitorService = monitorService

	// Scan the directory to populate folder info
	folderInfo, err := testDirService.ScanDirectory()
	require.NoError(t, err)
	appService.SetFolderInfo(folderInfo)

	cleanup := func() {
		if appService.P2PService != nil {
			appService.P2PService.Close()
		}
		if appService.DatabaseService != nil {
			appService.DatabaseService.Close()
		}
		if appService.MonitorService != nil {
			appService.MonitorService.Stop()
		}
	}

	t.Logf("✅ Test node '%s' created successfully at %s", nodeName, tempDir)
	return appService, cleanup
}

// testDirectoryService is a wrapper around DirectoryService for testing
type testDirectoryService struct {
	baseDir string
}

func (t *testDirectoryService) CreateDirectory() error {
	return os.MkdirAll(filepath.Join(t.baseDir, "space184"), 0755)
}

func (t *testDirectoryService) ScanDirectory() (*models.FolderInfo, error) {
	space184Path := filepath.Join(t.baseDir, "space184")
	entries, err := os.ReadDir(space184Path)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return &models.FolderInfo{
		Path:     space184Path,
		Files:    files,
		LastScan: time.Now(),
	}, nil
}

func (t *testDirectoryService) GetDirectoryPath() string {
	return filepath.Join(t.baseDir, "space184")
}

// extractIPAndPort extracts IP and port from a multiaddr string
func extractIPAndPort(addr string) (string, int) {
	// Parse multiaddr format: /ip4/x.x.x.x/tcp/port
	if len(addr) > 5 && addr[:5] == "/ip4/" {
		parts := make([]string, 0)
		current := ""
		for i := 1; i < len(addr); i++ {
			if addr[i] == '/' {
				if current != "" {
					parts = append(parts, current)
					current = ""
				}
			} else {
				current += string(addr[i])
			}
		}
		if current != "" {
			parts = append(parts, current)
		}

		if len(parts) >= 4 && parts[0] == "ip4" && parts[2] == "tcp" {
			ip := parts[1]
			port := 0
			if _, err := fmt.Sscanf(parts[3], "%d", &port); err == nil {
				return ip, port
			}
		}
	}
	return "", 0
}