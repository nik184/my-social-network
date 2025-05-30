package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/multiformats/go-multiaddr"

	"my-social-network/internal/models"
)

const (
	// Protocol ID for our application
	AppProtocol = "/my-social-network/1.0.0"

	// Service tag for mDNS discovery
	ServiceTag = "my-social-network-p2p"

	// Application identifier for peer validation
	AppIdentifier = "MySocialNetwork-DistributedApp"

	// Protocol for peer identification
	IdentifyProtocol = "/my-social-network/identify/1.0.0"

	// Protocol for rendezvous/relay assistance
	RendezvousProtocol = "/my-social-network/rendezvous/1.0.0"

	// NAT traversal assistance protocol
	NATAssistProtocol = "/my-social-network/nat-assist/1.0.0"
)

// PeerInfo stores information about connected peers
type PeerInfo struct {
	ID             peer.ID   `json:"id"`
	Addresses      []string  `json:"addresses"`
	FirstSeen      time.Time `json:"first_seen"`
	LastSeen       time.Time `json:"last_seen"`
	IsValidated    bool      `json:"is_validated"`
	ConnectionType string    `json:"connection_type"` // "inbound" or "outbound"
	Name           string    `json:"name"`            // peer's display name
}

// P2PService handles libp2p networking
type P2PService struct {
	host           host.Host
	dht            *dht.IpfsDHT
	ctx            context.Context
	cancel         context.CancelFunc
	appService     *AppService
	dbService      *DatabaseService
	validatedPeers map[peer.ID]bool
	peersMutex     sync.RWMutex

	// NAT detection and relay assistance
	isPublicNode   bool
	natDetected    bool
	connectedPeers map[peer.ID]*PeerInfo
	peerInfoMutex  sync.RWMutex
}

// NewP2PService creates a new P2P service
func NewP2PService(appService *AppService, dbService *DatabaseService) (*P2PService, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Get persistent private key from database
	privateKey, err := dbService.GetNodePrivateKey()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to get node private key: %w", err)
	}

	// Connection manager to handle connection limits
	connmgr, err := connmgr.NewConnManager(
		10,  // Lowwater
		100, // HighWater
		connmgr.WithGracePeriod(time.Minute),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}

	// Find available ports for P2P communication
	tcpPort, err := FindAvailablePort(9000)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to find available TCP port: %w", err)
	}

	quicPort, err := FindAvailablePort(tcpPort + 1)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to find available QUIC port: %w", err)
	}

	log.Printf("🔌 Using P2P ports - TCP: %d, QUIC: %d", tcpPort, quicPort)

	// Create libp2p host with persistent identity and available ports
	h, err := libp2p.New(
		libp2p.Identity(privateKey), // Use persistent private key
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", tcpPort),       // TCP on available port
			fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", quicPort), // QUIC on available port
		),
		libp2p.ConnectionManager(connmgr),
		libp2p.EnableHolePunching(), // Enable hole punching
		libp2p.EnableNATService(),   // Enable NAT service
		libp2p.DefaultSecurity,      // Use default security protocols
		libp2p.DefaultMuxers,        // Use default stream multiplexers
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	log.Printf("🚀 P2P Host created successfully!")
	log.Printf("📋 Peer ID: %s", h.ID())
	log.Printf("🌐 Listening on addresses:")
	for _, addr := range h.Addrs() {
		log.Printf("   %s/p2p/%s", addr, h.ID())
	}
	log.Printf("🔧 Features enabled: Hole Punching, NAT Service, DHT Discovery")

	service := &P2PService{
		host:           h,
		ctx:            ctx,
		cancel:         cancel,
		appService:     appService,
		dbService:      dbService,
		validatedPeers: make(map[peer.ID]bool),
		connectedPeers: make(map[peer.ID]*PeerInfo),
	}

	// Set stream handler for our protocol
	h.SetStreamHandler(protocol.ID(AppProtocol), service.handleStream)
	h.SetStreamHandler(protocol.ID(IdentifyProtocol), service.handleIdentifyStream)
	h.SetStreamHandler(protocol.ID(RendezvousProtocol), service.handleRendezvousStream)
	h.SetStreamHandler(protocol.ID(NATAssistProtocol), service.handleNATAssistStream)

	// Detect NAT status
	service.detectNATStatus()

	// Initialize DHT for global peer discovery
	if err := service.setupDHT(); err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("failed to setup DHT: %w", err)
	}

	// Setup mDNS for local network discovery
	if err := service.setupMDNS(); err != nil {
		log.Printf("Warning: mDNS setup failed: %v", err)
	}

	return service, nil
}

// setupDHT initializes the DHT for global peer discovery
func (p *P2PService) setupDHT() error {
	// Create DHT
	kadDHT, err := dht.New(p.ctx, p.host)
	if err != nil {
		return fmt.Errorf("failed to create DHT: %w", err)
	}
	p.dht = kadDHT

	// Bootstrap the DHT
	if err := kadDHT.Bootstrap(p.ctx); err != nil {
		return fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	// Skip IPFS bootstrap nodes to avoid connecting to non-app peers
	log.Printf("🔍 DHT initialized - using local peer discovery only")

	// Setup relay discovery after DHT is ready
	go p.setupRelayDiscovery()

	// Start periodic peer cleanup
	go p.startPeerCleanup()

	return nil
}

// setupRelayDiscovery sets up relay discovery using the DHT
func (p *P2PService) setupRelayDiscovery() {
	// Wait a bit for DHT to be ready
	time.Sleep(5 * time.Second)

	// Look for peers that might act as relays
	// This is a simplified approach - in production you'd want more sophisticated relay discovery
	peers := p.host.Network().Peers()
	log.Printf("Current connected peers: %d", len(peers))

	if len(peers) > 0 {
		log.Printf("Connected to %d peers, relay functionality available through DHT", len(peers))
	} else {
		log.Printf("No peers connected yet, will rely on bootstrap nodes for initial connectivity")
	}
}

// handleIdentifyStream handles peer identification requests
func (p *P2PService) handleIdentifyStream(stream network.Stream) {
	defer stream.Close()

	peerID := stream.Conn().RemotePeer()
	log.Printf("🔍 Received identification request from peer: %s", peerID)

	// Read the requesting peer's identification data (client sends first)
	var peerRequest map[string]string
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&peerRequest); err != nil {
		log.Printf("Failed to decode peer identification: %v", err)
		return
	}

	// Get our node name from database
	nodeName := "unknown"
	if p.dbService != nil {
		if name, err := p.dbService.GetSetting("name"); err == nil {
			nodeName = name
		}
	}

	// Send our application identifier response including name
	response := map[string]string{
		"app":     AppIdentifier,
		"version": "1.0.0",
		"nodeId":  p.host.ID().String(),
		"name":    nodeName,
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(response); err != nil {
		log.Printf("Failed to send identification response: %v", err)
		return
	}

	// Extract peer name and validate it's our application
	if app, exists := peerRequest["app"]; exists && app == AppIdentifier {
		peerName := "unknown"
		if name, exists := peerRequest["name"]; exists {
			peerName = name
		}

		// Mark peer as validated and save connection with name
		p.markPeerValidationWithName(peerID, true, peerName)
		log.Printf("✅ Validated incoming peer: %s (name: %s)", peerID, peerName)
	}

	log.Printf("✅ Sent identification response to peer: %s (name: %s)", peerID, nodeName)
}

// validatePeer checks if a peer is running our application
func (p *P2PService) validatePeer(peerID peer.ID) bool {
	// Check if already validated
	p.peersMutex.RLock()
	if validated, exists := p.validatedPeers[peerID]; exists {
		p.peersMutex.RUnlock()
		return validated
	}
	p.peersMutex.RUnlock()

	// Don't validate ourselves
	if peerID == p.host.ID() {
		return false
	}

	log.Printf("🔍 Validating peer: %s", peerID)

	// Try to open identification stream
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()

	stream, err := p.host.NewStream(ctx, peerID, protocol.ID(IdentifyProtocol))
	if err != nil {
		log.Printf("❌ Failed to open identification stream to %s: %v", peerID, err)
		p.markPeerValidation(peerID, false)
		return false
	}
	defer stream.Close()

	// As the stream initiator (client), send our identification data first
	ourNodeName := "unknown"
	if p.dbService != nil {
		if name, err := p.dbService.GetSetting("name"); err == nil {
			ourNodeName = name
		}
	}

	ourRequest := map[string]string{
		"app":     AppIdentifier,
		"version": "1.0.0",
		"nodeId":  p.host.ID().String(),
		"name":    ourNodeName,
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(ourRequest); err != nil {
		log.Printf("❌ Failed to send our identification to %s: %v", peerID, err)
		p.markPeerValidation(peerID, false)
		return false
	}

	// Read identification response from remote peer
	var response map[string]string
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&response); err != nil {
		log.Printf("❌ Failed to decode identification from %s: %v", peerID, err)
		p.markPeerValidation(peerID, false)
		return false
	}

	// Check if it's our application
	if app, exists := response["app"]; !exists || app != AppIdentifier {
		log.Printf("❌ Peer %s is not running our application (app: %s)", peerID, app)
		p.markPeerValidation(peerID, false)
		return false
	}

	// Extract peer name if available
	peerName := "unknown"
	if name, exists := response["name"]; exists {
		peerName = name
	}

	log.Printf("✅ Peer %s validated as our application (name: %s)", peerID, peerName)
	p.markPeerValidationWithName(peerID, true, peerName)
	return true
}

// markPeerValidation marks a peer as validated or not
func (p *P2PService) markPeerValidation(peerID peer.ID, isValid bool) {
	p.markPeerValidationWithName(peerID, isValid, "")
}

// markPeerValidationWithName marks a peer as validated with name information
func (p *P2PService) markPeerValidationWithName(peerID peer.ID, isValid bool, peerName string) {
	p.peersMutex.Lock()
	p.validatedPeers[peerID] = isValid
	p.peersMutex.Unlock()

	// Update peer info validation status and name
	p.peerInfoMutex.Lock()
	if peerInfo, exists := p.connectedPeers[peerID]; exists {
		peerInfo.IsValidated = isValid
		if peerName != "" {
			peerInfo.Name = peerName
		}

		// Update validation status and name in database
		if p.dbService != nil && len(peerInfo.Addresses) > 0 {
			address := peerInfo.Addresses[0] // Use first address
			if err := p.dbService.RecordConnectionWithName(peerID.String(), address, peerInfo.ConnectionType, isValid, peerName); err != nil {
				log.Printf("Warning: Failed to update validation status in database: %v", err)
			}
		}
	}
	p.peerInfoMutex.Unlock()
}

// setupMDNS initializes mDNS for local network discovery
func (p *P2PService) setupMDNS() error {
	// Setup mDNS discovery service
	service := mdns.NewMdnsService(p.host, ServiceTag, &discoveryNotifee{p2pService: p})
	return service.Start()
}

// discoveryNotifee handles peer discovery notifications
type discoveryNotifee struct {
	p2pService *P2PService
}

func (n *discoveryNotifee) HandlePeerFound(peerInfo peer.AddrInfo) {
	log.Printf("🔍 Discovered peer via mDNS: %s", peerInfo.ID)

	// Connect to discovered peer
	ctx, cancel := context.WithTimeout(n.p2pService.ctx, 5*time.Second)
	defer cancel()

	if err := n.p2pService.host.Connect(ctx, peerInfo); err != nil {
		log.Printf("❌ Failed to connect to discovered peer %s: %v", peerInfo.ID, err)
		return
	}

	log.Printf("🔗 Connected to peer: %s", peerInfo.ID)

	// Store peer info and validate that this peer is running our application
	go func() {
		// Store peer information
		n.p2pService.storePeerInfo(peerInfo.ID, "outbound")

		// Give the connection a moment to stabilize
		time.Sleep(1 * time.Second)

		if n.p2pService.validatePeer(peerInfo.ID) {
			log.Printf("✅ mDNS peer %s validated as our application", peerInfo.ID)
		} else {
			log.Printf("❌ mDNS peer %s is not our application, disconnecting", peerInfo.ID)
			n.p2pService.host.Network().ClosePeer(peerInfo.ID)
		}
	}()
}

// handleStream handles incoming streams
func (p *P2PService) handleStream(stream network.Stream) {
	defer stream.Close()

	peerID := stream.Conn().RemotePeer()

	// Store peer information for incoming connections
	p.storePeerInfo(peerID, "inbound")

	var msg models.P2PMessage
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&msg); err != nil {
		log.Printf("Failed to decode message: %v", err)
		return
	}

	log.Printf("Received message type: %s from %s", msg.Type, peerID)

	var response models.P2PMessage

	switch msg.Type {
	case models.MessageTypeGetInfo:
		// Return node and folder information
		response = models.P2PMessage{
			Type:    models.MessageTypeGetInfoResp,
			Payload: p.appService.GetNodeInfo(),
		}

	case models.MessageTypeDiscovery:
		// Handle discovery request
		response = models.P2PMessage{
			Type:    models.MessageTypeDiscoveryResp,
			Payload: p.appService.GetNodeInfo(),
		}

	case models.MessageTypeGetPeerList:
		// Return list of connected peers
		log.Printf("📋 Processing peer list request from %s", peerID)
		peerList := p.getConnectedPeersList()
		response = models.P2PMessage{
			Type:    models.MessageTypeGetPeerListResp,
			Payload: peerList,
		}
		log.Printf("📋 Prepared peer list response with %d peers for %s", peerList.Count, peerID)

	case models.MessageTypeHolePunchAssist:
		// Handle hole punching assistance request
		assistResponse := p.handleHolePunchAssistRequest(msg.Payload)
		response = models.P2PMessage{
			Type:    models.MessageTypeHolePunchResp,
			Payload: assistResponse,
		}

	default:
		log.Printf("Unknown message type: %s", msg.Type)
		return
	}

	// Send response
	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(response); err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}

// GetNode returns current network node information
func (p *P2PService) GetNode() *models.NetworkNode {
	return &models.NetworkNode{
		ID:        p.host.ID(),
		Addresses: p.host.Addrs(),
		LastSeen:  time.Now(),
	}
}

// DiscoverPeer attempts to discover and communicate with a peer
func (p *P2PService) DiscoverPeer(peerID string) (*models.NodeInfoResponse, error) {
	// Parse peer ID
	pid, err := peer.Decode(peerID)
	if err != nil {
		return nil, fmt.Errorf("invalid peer ID: %w", err)
	}

	// Try to find peer in DHT
	ctx, cancel := context.WithTimeout(p.ctx, 30*time.Second)
	defer cancel()

	peerInfo, err := p.dht.FindPeer(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("failed to find peer in DHT: %w", err)
	}

	log.Printf("Found peer %s at addresses: %v", pid, peerInfo.Addrs)

	// Connect to peer if not already connected
	if p.host.Network().Connectedness(pid) != network.Connected {
		if err := p.host.Connect(ctx, peerInfo); err != nil {
			return nil, fmt.Errorf("failed to connect to peer: %w", err)
		}
	}

	// Validate that this peer is running our application
	if !p.validatePeer(pid) {
		return nil, fmt.Errorf("peer %s is not running our application", pid)
	}

	// Open stream to peer
	stream, err := p.host.NewStream(ctx, pid, protocol.ID(AppProtocol))
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}
	defer stream.Close()

	// Send discovery message
	msg := models.P2PMessage{
		Type:    models.MessageTypeGetInfo,
		Payload: nil,
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(msg); err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Read response
	var response models.P2PMessage
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Parse response payload
	responseData, err := json.Marshal(response.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response payload: %w", err)
	}

	var nodeInfo models.NodeInfoResponse
	if err := json.Unmarshal(responseData, &nodeInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node info: %w", err)
	}

	return &nodeInfo, nil
}

// GetConnectedPeers returns list of validated connected peers
func (p *P2PService) GetConnectedPeers() []peer.ID {
	allPeers := p.host.Network().Peers()
	var validatedPeersList []peer.ID

	p.peersMutex.RLock()
	defer p.peersMutex.RUnlock()

	for _, peerID := range allPeers {
		// Check if peer is validated as our application
		if validated, exists := p.validatedPeers[peerID]; exists && validated {
			validatedPeersList = append(validatedPeersList, peerID)
		} else if !exists {
			// Trigger validation for unknown peers
			go p.validatePeer(peerID)
		}
	}

	return validatedPeersList
}

// GetAllConnectedPeers returns list of all connected peers (validated and unvalidated)
func (p *P2PService) GetAllConnectedPeers() []peer.ID {
	return p.host.Network().Peers()
}

// startPeerCleanup runs periodic cleanup of invalid peers
func (p *P2PService) startPeerCleanup() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.cleanupInvalidPeers()
		}
	}
}

// cleanupInvalidPeers disconnects from peers that failed validation
func (p *P2PService) cleanupInvalidPeers() {
	allPeers := p.host.Network().Peers()
	var disconnectedCount int

	p.peersMutex.RLock()
	for _, peerID := range allPeers {
		if validated, exists := p.validatedPeers[peerID]; exists && !validated {
			// Disconnect from invalid peer
			p.host.Network().ClosePeer(peerID)
			disconnectedCount++
			log.Printf("🧹 Disconnected from invalid peer: %s", peerID)
		}
	}
	p.peersMutex.RUnlock()

	if disconnectedCount > 0 {
		log.Printf("🧹 Cleanup complete: disconnected from %d invalid peers", disconnectedCount)
	}
}

// detectNATStatus determines if this node is publicly accessible
func (p *P2PService) detectNATStatus() {
	p.natDetected = true
	p.isPublicNode = false

	// Check if any of our listening addresses are public
	for _, addr := range p.host.Addrs() {
		addrStr := addr.String()

		// Extract IP from multiaddr (format: /ip4/x.x.x.x/tcp/port)
		if ip := extractIPFromMultiaddr(addrStr); ip != nil {
			if isPublicIP(ip) {
				p.isPublicNode = true
				log.Printf("🌐 Detected as PUBLIC node - can assist with NAT traversal")
				log.Printf("   Public address: %s", addrStr)
				return
			}
		}
	}

	log.Printf("🏠 Detected as NAT'd node - will seek assistance for connections")
}

// extractIPFromMultiaddr extracts IP address from multiaddr string
func extractIPFromMultiaddr(addrStr string) net.IP {
	// Simple extraction for /ip4/x.x.x.x/... format
	if len(addrStr) > 5 && addrStr[:5] == "/ip4/" {
		parts := []rune(addrStr[5:])
		var ipStr string
		for _, r := range parts {
			if r == '/' {
				break
			}
			ipStr += string(r)
		}
		return net.ParseIP(ipStr)
	}
	return nil
}

// isPublicIP checks if an IP address is publicly routable
func isPublicIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}

	// Check for private IP ranges
	if ip4 := ip.To4(); ip4 != nil {
		// 10.0.0.0/8
		if ip4[0] == 10 {
			return false
		}
		// 172.16.0.0/12
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return false
		}
		// 192.168.0.0/16
		if ip4[0] == 192 && ip4[1] == 168 {
			return false
		}
	}

	return true
}

// handleRendezvousStream handles rendezvous/relay assistance requests
func (p *P2PService) handleRendezvousStream(stream network.Stream) {
	defer stream.Close()

	if !p.isPublicNode {
		log.Printf("⚠️ Received rendezvous request but this node is not public")
		return
	}

	peerID := stream.Conn().RemotePeer()
	log.Printf("🤝 Handling rendezvous request from peer: %s", peerID)

	var request map[string]interface{}
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&request); err != nil {
		log.Printf("Failed to decode rendezvous request: %v", err)
		return
	}

	// Store peer information for assistance
	p.storePeerInfo(peerID, "inbound")

	// Send response with our peer information and known peers
	response := map[string]interface{}{
		"status":      "success",
		"public_node": true,
		"peer_id":     p.host.ID().String(),
		"known_peers": p.getKnownPeersList(),
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(response); err != nil {
		log.Printf("Failed to send rendezvous response: %v", err)
	}

	log.Printf("✅ Rendezvous response sent to peer: %s", peerID)
}

// handleNATAssistStream handles NAT traversal assistance requests
func (p *P2PService) handleNATAssistStream(stream network.Stream) {
	defer stream.Close()

	if !p.isPublicNode {
		log.Printf("⚠️ Received NAT assist request but this node is not public")
		return
	}

	peerID := stream.Conn().RemotePeer()
	log.Printf("🔧 Handling NAT assistance request from peer: %s", peerID)

	var request map[string]interface{}
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&request); err != nil {
		log.Printf("Failed to decode NAT assist request: %v", err)
		return
	}

	// Extract target peer ID for hole punching assistance
	targetPeerStr, ok := request["target_peer"].(string)
	if !ok {
		log.Printf("Invalid NAT assist request: missing target_peer")
		return
	}

	targetPeer, err := peer.Decode(targetPeerStr)
	if err != nil {
		log.Printf("Invalid target peer ID: %v", err)
		return
	}

	// Check if we know the target peer
	p.peerInfoMutex.RLock()
	targetInfo, exists := p.connectedPeers[targetPeer]
	p.peerInfoMutex.RUnlock()

	response := map[string]interface{}{
		"status":       "success",
		"target_known": exists,
	}

	if exists {
		response["target_addresses"] = targetInfo.Addresses
		log.Printf("🎯 Providing address information for target peer: %s", targetPeer)
	} else {
		log.Printf("❓ Target peer %s not known to this relay", targetPeer)
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(response); err != nil {
		log.Printf("Failed to send NAT assist response: %v", err)
	}
}

// storePeerInfo stores information about a connected peer
func (p *P2PService) storePeerInfo(peerID peer.ID, connectionType string) {
	if peerID == p.host.ID() {
		return // Don't store info about ourselves
	}

	p.peerInfoMutex.Lock()
	defer p.peerInfoMutex.Unlock()

	now := time.Now()
	peerInfo, exists := p.connectedPeers[peerID]

	// Get peer address
	var address string
	if conn := p.host.Network().ConnsToPeer(peerID); len(conn) > 0 {
		address = conn[0].RemoteMultiaddr().String()
	}

	if !exists {
		// Create new peer info
		addrs := make([]string, 0)
		if address != "" {
			addrs = append(addrs, address)
		}

		peerInfo = &PeerInfo{
			ID:             peerID,
			Addresses:      addrs,
			FirstSeen:      now,
			LastSeen:       now,
			ConnectionType: connectionType,
		}
		p.connectedPeers[peerID] = peerInfo

		if p.isPublicNode {
			log.Printf("📝 Stored peer info: %s (%s connection)", peerID, connectionType)
		}
	} else {
		// Update existing peer info
		peerInfo.LastSeen = now
		if connectionType != "" && peerInfo.ConnectionType == "" {
			peerInfo.ConnectionType = connectionType
		}
	}

	// Record connection in database
	if p.dbService != nil && address != "" {
		if err := p.dbService.RecordConnection(peerID.String(), address, connectionType, false); err != nil {
			log.Printf("Warning: Failed to record connection in database: %v", err)
		}
	}
}

// getKnownPeersList returns a list of known peer IDs
func (p *P2PService) getKnownPeersList() []string {
	p.peerInfoMutex.RLock()
	defer p.peerInfoMutex.RUnlock()

	peers := make([]string, 0, len(p.connectedPeers))
	for peerID := range p.connectedPeers {
		peers = append(peers, peerID.String())
	}

	return peers
}

// IsPublicNode returns whether this node can assist with NAT traversal
func (p *P2PService) IsPublicNode() bool {
	return p.isPublicNode
}

// GetConnectedPeerInfo returns detailed information about connected peers
func (p *P2PService) GetConnectedPeerInfo() map[peer.ID]*PeerInfo {
	p.peerInfoMutex.RLock()
	defer p.peerInfoMutex.RUnlock()

	// Create a copy to avoid race conditions
	result := make(map[peer.ID]*PeerInfo)
	for id, info := range p.connectedPeers {
		infoCopy := *info
		result[id] = &infoCopy
	}

	return result
}

// ConnectByIP connects to a peer using IP address and port
func (p *P2PService) ConnectByIP(ip string, port int, peerIDStr string) (*models.NodeInfoResponse, error) {
	// Parse peer ID
	pid, err := peer.Decode(peerIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid peer ID: %w", err)
	}

	// Create multiaddr from IP and port
	addr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, port))
	if err != nil {
		return nil, fmt.Errorf("failed to create multiaddr: %w", err)
	}

	// Create peer info
	peerInfo := peer.AddrInfo{
		ID:    pid,
		Addrs: []multiaddr.Multiaddr{addr},
	}

	log.Printf("🌐 Attempting to connect to peer %s at %s:%d", pid, ip, port)

	// Connect to peer with timeout
	ctx, cancel := context.WithTimeout(p.ctx, 30*time.Second)
	defer cancel()

	if err := p.host.Connect(ctx, peerInfo); err != nil {
		// Provide more helpful error messages
		errStr := err.Error()
		if strings.Contains(errStr, "failed to negotiate security protocol") {
			return nil, fmt.Errorf("failed to connect to peer at %s:%d: Protocol mismatch - make sure you're using the P2P port (not web port). Original error: %w", ip, port, err)
		}
		if strings.Contains(errStr, "connection refused") {
			return nil, fmt.Errorf("failed to connect to peer at %s:%d: Connection refused - check if the node is running and port %d is open in firewall. Original error: %w", ip, port, port, err)
		}
		if strings.Contains(errStr, "timeout") {
			return nil, fmt.Errorf("failed to connect to peer at %s:%d: Connection timeout - check network connectivity and firewall settings. Original error: %w", ip, port, err)
		}
		return nil, fmt.Errorf("failed to connect to peer at %s:%d: %w", ip, port, err)
	}

	log.Printf("✅ Successfully connected to peer %s at %s:%d", pid, ip, port)

	// Store peer information
	p.storePeerInfo(pid, "outbound")

	// Validate that this peer is running our application
	if !p.validatePeer(pid) {
		log.Printf("❌ Peer %s is not running our application, disconnecting", pid)
		p.host.Network().ClosePeer(pid)
		return nil, fmt.Errorf("peer %s is not running our application", pid)
	}

	// Open stream to get node info
	stream, err := p.host.NewStream(ctx, pid, protocol.ID(AppProtocol))
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}
	defer stream.Close()

	// Send discovery message
	msg := models.P2PMessage{
		Type:    models.MessageTypeGetInfo,
		Payload: nil,
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(msg); err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Read response
	var response models.P2PMessage
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Parse response payload
	responseData, err := json.Marshal(response.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response payload: %w", err)
	}

	var nodeInfo models.NodeInfoResponse
	if err := json.Unmarshal(responseData, &nodeInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node info: %w", err)
	}

	log.Printf("📋 Received node info from peer %s", pid)
	return &nodeInfo, nil
}

// GetConnectionInfo returns connection information for sharing
func (p *P2PService) GetConnectionInfo() *models.ConnectionInfo {
	connectionInfo := &models.ConnectionInfo{
		PeerID:       p.host.ID().String(),
		IsPublicNode: p.isPublicNode,
	}

	// Get all listening addresses
	var localAddresses []string
	var publicAddress string
	var port int

	for _, addr := range p.host.Addrs() {
		addrStr := addr.String()
		localAddresses = append(localAddresses, addrStr)

		// Extract IP and port for public addresses
		if ip := extractIPFromMultiaddr(addrStr); ip != nil {
			if isPublicIP(ip) && publicAddress == "" {
				publicAddress = ip.String()
				// Extract port from multiaddr
				if portValue := extractPortFromMultiaddr(addrStr); portValue != 0 {
					port = portValue
				}
			}
		}
	}

	connectionInfo.LocalAddresses = localAddresses
	if publicAddress != "" && port != 0 {
		connectionInfo.PublicAddress = publicAddress
		connectionInfo.Port = port
	}

	return connectionInfo
}

// extractPortFromMultiaddr extracts port from multiaddr string
func extractPortFromMultiaddr(addrStr string) int {
	// Simple extraction for /ip4/x.x.x.x/tcp/port format
	parts := []rune(addrStr)
	var portStr string
	var inPort bool

	for i := 0; i < len(parts)-4; i++ {
		if string(parts[i:i+5]) == "/tcp/" {
			inPort = true
			i += 4
			continue
		}
		if inPort {
			if parts[i] == '/' {
				break
			}
			portStr += string(parts[i])
		}
	}

	if portStr != "" {
		if port, err := parsePort(portStr); err == nil {
			return port
		}
	}
	return 0
}

// parsePort converts string to int
func parsePort(portStr string) (int, error) {
	port := 0
	for _, r := range portStr {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid port character")
		}
		port = port*10 + int(r-'0')
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port out of range")
	}
	return port, nil
}

// getConnectedPeersList returns a list of connected validated peers
func (p *P2PService) getConnectedPeersList() models.PeerListResponse {
	connectedPeers := p.GetConnectedPeers()
	var peerList []models.PeerListItem

	log.Printf("📋 Preparing peer list response: %d connected peers", len(connectedPeers))

	for _, peerID := range connectedPeers {
		peerName := "unknown"

		// Get peer name from stored peer info
		p.peerInfoMutex.RLock()
		if peerInfo, exists := p.connectedPeers[peerID]; exists && peerInfo.Name != "" {
			peerName = peerInfo.Name
		}
		p.peerInfoMutex.RUnlock()

		peerList = append(peerList, models.PeerListItem{
			PeerID:   peerID.String(),
			PeerName: peerName,
		})
	}

	response := models.PeerListResponse{
		Peers: peerList,
		Count: len(peerList),
	}

	log.Printf("📋 Returning peer list with %d peers", response.Count)
	return response
}

// requestPeerListFromPeer requests the peer list from a connected peer
func (p *P2PService) requestPeerListFromPeer(peerID peer.ID) (*models.PeerListResponse, error) {
	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer cancel()

	// Open stream to peer
	stream, err := p.host.NewStream(ctx, peerID, protocol.ID(AppProtocol))
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}
	defer stream.Close()

	// Send peer list request
	msg := models.P2PMessage{
		Type:    models.MessageTypeGetPeerList,
		Payload: nil,
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(msg); err != nil {
		return nil, fmt.Errorf("failed to send peer list request: %w", err)
	}

	// Read response
	var response models.P2PMessage
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&response); err != nil {
		// Special handling for EOF - this might happen if peer has no connections
		if err == io.EOF {
			// Return empty peer list
			return &models.PeerListResponse{
				Peers: []models.PeerListItem{},
				Count: 0,
			}, nil
		}
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Type != models.MessageTypeGetPeerListResp {
		return nil, fmt.Errorf("unexpected response type: %s", response.Type)
	}

	// Parse response payload
	responseData, err := json.Marshal(response.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response payload: %w", err)
	}

	var peerListResponse models.PeerListResponse
	if err := json.Unmarshal(responseData, &peerListResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal peer list: %w", err)
	}

	return &peerListResponse, nil
}

// handleHolePunchAssistRequest handles hole punching assistance requests
func (p *P2PService) handleHolePunchAssistRequest(payload interface{}) map[string]interface{} {
	// Parse the request payload
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   "failed to parse request",
		}
	}

	var request map[string]string
	if err := json.Unmarshal(payloadData, &request); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   "invalid request format",
		}
	}

	targetPeerStr, exists := request["target_peer_id"]
	if !exists {
		return map[string]interface{}{
			"success": false,
			"error":   "missing target_peer_id",
		}
	}

	targetPeerID, err := peer.Decode(targetPeerStr)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   "invalid target peer ID",
		}
	}

	// Check if we're connected to the target peer
	p.peerInfoMutex.RLock()
	targetInfo, exists := p.connectedPeers[targetPeerID]
	p.peerInfoMutex.RUnlock()

	if !exists {
		return map[string]interface{}{
			"success": false,
			"error":   "target peer not connected to this node",
		}
	}

	// Return target peer's connection information
	return map[string]interface{}{
		"success":     true,
		"target_peer": targetPeerStr,
		"addresses":   targetInfo.Addresses,
		"name":        targetInfo.Name,
	}
}

// GetSecondDegreeConnections discovers peers connected to our direct connections
func (p *P2PService) GetSecondDegreeConnections() (*models.SecondDegreeConnectionsResponse, error) {
	connectedPeers := p.GetConnectedPeers()
	secondDegreePeers := make(map[string]models.SecondDegreePeer) // Use map to avoid duplicates

	// For each connected peer, request their peer list
	for _, peerID := range connectedPeers {
		viaPeerName := "unknown"

		// Get the via peer's name
		p.peerInfoMutex.RLock()
		if peerInfo, exists := p.connectedPeers[peerID]; exists && peerInfo.Name != "" {
			viaPeerName = peerInfo.Name
		}
		p.peerInfoMutex.RUnlock()

		// Request peer list from this peer
		peerList, err := p.requestPeerListFromPeer(peerID)
		if err != nil {
			log.Printf("Failed to get peer list from %s: %v", peerID, err)
			continue
		}

		// Process the peer list
		for _, remotePeer := range peerList.Peers {
			// Skip if it's ourselves
			if remotePeer.PeerID == p.host.ID().String() {
				continue
			}

			// Skip if we're already directly connected to this peer
			remotePeerID, err := peer.Decode(remotePeer.PeerID)
			if err != nil {
				continue
			}

			isDirectlyConnected := false
			for _, directPeer := range connectedPeers {
				if directPeer == remotePeerID {
					isDirectlyConnected = true
					break
				}
			}

			if isDirectlyConnected {
				continue
			}

			// Add to second-degree peers (using peer ID as key to avoid duplicates)
			secondDegreePeers[remotePeer.PeerID] = models.SecondDegreePeer{
				PeerID:      remotePeer.PeerID,
				PeerName:    remotePeer.PeerName,
				ViaPeerID:   peerID.String(),
				ViaPeerName: viaPeerName,
			}
		}
	}

	// Convert map to slice
	var peerList []models.SecondDegreePeer
	for _, peer := range secondDegreePeers {
		peerList = append(peerList, peer)
	}

	return &models.SecondDegreeConnectionsResponse{
		Peers: peerList,
		Count: len(peerList),
	}, nil
}

// ConnectToSecondDegreePeer attempts to connect to a second-degree peer using hole punching
func (p *P2PService) ConnectToSecondDegreePeer(targetPeerID, viaPeerID string) (*models.NodeInfoResponse, error) {
	// Parse peer IDs
	targetPeer, err := peer.Decode(targetPeerID)
	if err != nil {
		return nil, fmt.Errorf("invalid target peer ID: %w", err)
	}

	viaPeer, err := peer.Decode(viaPeerID)
	if err != nil {
		return nil, fmt.Errorf("invalid via peer ID: %w", err)
	}

	log.Printf("🔧 Attempting hole punch connection to %s via %s", targetPeer, viaPeer)

	// First, request hole punch assistance from the via peer
	ctx, cancel := context.WithTimeout(p.ctx, 30*time.Second)
	defer cancel()

	stream, err := p.host.NewStream(ctx, viaPeer, protocol.ID(AppProtocol))
	if err != nil {
		return nil, fmt.Errorf("failed to open stream to via peer: %w", err)
	}
	defer stream.Close()

	// Send hole punch assistance request
	assistRequest := models.P2PMessage{
		Type: models.MessageTypeHolePunchAssist,
		Payload: map[string]string{
			"target_peer_id": targetPeerID,
		},
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(assistRequest); err != nil {
		return nil, fmt.Errorf("failed to send hole punch request: %w", err)
	}

	// Read response
	var response models.P2PMessage
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode hole punch response: %w", err)
	}

	// Parse the response
	responseData, err := json.Marshal(response.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response payload: %w", err)
	}

	var assistResponse map[string]interface{}
	if err := json.Unmarshal(responseData, &assistResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal assist response: %w", err)
	}

	success, exists := assistResponse["success"].(bool)
	if !exists || !success {
		errorMsg := "hole punch assistance failed"
		if errStr, exists := assistResponse["error"].(string); exists {
			errorMsg = errStr
		}
		return nil, fmt.Errorf("hole punch assistance failed: %s", errorMsg)
	}

	// Extract target peer addresses
	addresses, exists := assistResponse["addresses"].([]interface{})
	if !exists || len(addresses) == 0 {
		return nil, fmt.Errorf("no addresses provided for target peer")
	}

	// Try to connect using the provided addresses
	var lastErr error
	for _, addrInterface := range addresses {
		addrStr, ok := addrInterface.(string)
		if !ok {
			continue
		}

		// Parse multiaddr
		addr, err := multiaddr.NewMultiaddr(addrStr)
		if err != nil {
			lastErr = err
			continue
		}

		// Create peer info
		peerInfo := peer.AddrInfo{
			ID:    targetPeer,
			Addrs: []multiaddr.Multiaddr{addr},
		}

		// Attempt connection
		connectCtx, connectCancel := context.WithTimeout(ctx, 15*time.Second)
		err = p.host.Connect(connectCtx, peerInfo)
		connectCancel()

		if err == nil {
			log.Printf("✅ Successfully connected to %s via hole punching", targetPeer)

			// Store peer information
			p.storePeerInfo(targetPeer, "outbound")

			// Validate the peer
			if !p.validatePeer(targetPeer) {
				log.Printf("❌ Peer %s is not running our application, disconnecting", targetPeer)
				p.host.Network().ClosePeer(targetPeer)
				return nil, fmt.Errorf("peer %s is not running our application", targetPeer)
			}

			// Get node info from the newly connected peer
			return p.DiscoverPeer(targetPeerID)
		}

		lastErr = err
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to connect to target peer: %w", lastErr)
	}

	return nil, fmt.Errorf("no valid addresses to connect to")
}

// Close shuts down the P2P service
func (p *P2PService) Close() error {
	p.cancel()
	if p.dht != nil {
		p.dht.Close()
	}
	return p.host.Close()
}
