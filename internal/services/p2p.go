package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
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

	"old-school/internal/interfaces"
	"old-school/internal/models"
)

const (
	// Protocol ID for our application
	AppProtocol = "/old-school/1.0.0"

	// Service tag for mDNS discovery
	ServiceTag = "old-school-p2p"

	// Application identifier for peer validation
	AppIdentifier = "MySocialNetwork-DistributedApp"

	// Protocol for peer identification
	IdentifyProtocol = "/old-school/identify/1.0.0"

	// Protocol for rendezvous/relay assistance
	RendezvousProtocol = "/old-school/rendezvous/1.0.0"

	// NAT traversal assistance protocol
	NATAssistProtocol = "/old-school/nat-assist/1.0.0"
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
	HasAvatar      bool      `json:"has_avatar"`      // whether peer has avatar images
}

// AvatarData represents avatar image data for peer identification
type AvatarData struct {
	Filename string `json:"filename"`
	Data     string `json:"data"` // base64 encoded image data
	Size     int    `json:"size"`
}

// P2PService handles libp2p networking
type P2PService struct {
	host           host.Host
	dht            *dht.IpfsDHT
	ctx            context.Context
	cancel         context.CancelFunc
	container      *ServiceContainer
	dbService      interfaces.DatabaseService
	validatedPeers map[peer.ID]bool
	peersMutex     sync.RWMutex

	// NAT detection and relay assistance
	isPublicNode   bool
	natDetected    bool
	connectedPeers map[peer.ID]*PeerInfo
	peerInfoMutex  sync.RWMutex
}

// NewP2PService creates a new P2P service
func NewP2PService(container *ServiceContainer, dbService interfaces.DatabaseService) (*P2PService, error) {
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
		container:      container,
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

	// Start periodic peer data retry check
	go p.startPeerDataRetry()

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

// prepareAvatarData reads the primary avatar image and encodes it for transmission
func (p *P2PService) prepareAvatarData() *AvatarData {
	if p.container == nil || p.container.GetDirectoryService() == nil {
		return nil
	}

	// Get avatar images
	avatarImages, err := p.container.GetDirectoryService().GetAvatarImages()
	if err != nil || len(avatarImages) == 0 {
		return nil
	}

	// Use the first image as the primary avatar
	primaryAvatar := avatarImages[0]
	avatarDir := p.container.GetDirectoryService().GetAvatarDirectory()
	avatarPath := filepath.Join(avatarDir, primaryAvatar)

	// Read the image file
	imageData, err := os.ReadFile(avatarPath)
	if err != nil {
		log.Printf("Failed to read avatar image %s: %v", primaryAvatar, err)
		return nil
	}

	// Limit avatar size to 1MB for transmission
	if len(imageData) > 1024*1024 {
		log.Printf("Avatar image %s too large (%d bytes), skipping", primaryAvatar, len(imageData))
		return nil
	}

	// Encode to base64
	encodedData := base64.StdEncoding.EncodeToString(imageData)

	return &AvatarData{
		Filename: primaryAvatar,
		Data:     encodedData,
		Size:     len(imageData),
	}
}

// prepareFriendsData prepares the current user's friends list for transmission
func (p *P2PService) prepareFriendsData() []map[string]interface{} {
	if p.dbService == nil {
		return nil
	}

	// Get the current user's friends
	friends, err := p.dbService.GetFriends()
	if err != nil {
		log.Printf("Failed to get friends for transmission: %v", err)
		return nil
	}

	if len(friends) == 0 {
		return nil
	}

	// Convert friends to transmission format
	friendsData := make([]map[string]interface{}, 0, len(friends))
	for _, friend := range friends {
		friendData := map[string]interface{}{
			"peer_id":   friend.PeerID,
			"peer_name": friend.PeerName,
		}
		friendsData = append(friendsData, friendData)
	}

	log.Printf("📤 Prepared %d friends for transmission", len(friendsData))
	return friendsData
}

// saveReceivedAvatar saves avatar data received from a peer
func (p *P2PService) saveReceivedAvatar(peerID peer.ID, avatarData *AvatarData) error {
	if p.container == nil || p.container.GetDirectoryService() == nil || avatarData == nil {
		return fmt.Errorf("invalid service or avatar data")
	}

	// Decode base64 data
	imageData, err := base64.StdEncoding.DecodeString(avatarData.Data)
	if err != nil {
		return fmt.Errorf("failed to decode avatar data: %w", err)
	}

	// Verify size matches
	if len(imageData) != avatarData.Size {
		return fmt.Errorf("avatar data size mismatch: expected %d, got %d", avatarData.Size, len(imageData))
	}

	// Save using DirectoryService
	err = p.container.GetDirectoryService().SavePeerAvatar(peerID.String(), avatarData.Filename, imageData)
	if err != nil {
		return fmt.Errorf("failed to save peer avatar: %w", err)
	}

	log.Printf("✅ Saved avatar %s for peer %s (%d bytes)", avatarData.Filename, peerID, len(imageData))
	return nil
}

// handleIdentifyStream handles peer identification requests
func (p *P2PService) handleIdentifyStream(stream network.Stream) {
	defer stream.Close()

	peerID := stream.Conn().RemotePeer()
	log.Printf("🔍 Received identification request from peer: %s", peerID)

	// Read the requesting peer's identification data (client sends first)
	var peerRequest map[string]interface{}
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

	// Prepare our avatar data for transmission
	avatarData := p.prepareAvatarData()

	// Prepare our friends data for transmission
	friendsData := p.prepareFriendsData()

	// Send our application identifier response including name and avatar
	response := map[string]interface{}{
		"app":     AppIdentifier,
		"version": "1.0.0",
		"nodeId":  p.host.ID().String(),
		"name":    nodeName,
	}

	// Include avatar data if available
	if avatarData != nil {
		response["avatar"] = avatarData
	}

	// Include friends data if available
	if friendsData != nil {
		response["friends"] = friendsData
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
			if nameStr, ok := name.(string); ok {
				peerName = nameStr
			}
		}

		// Process avatar data if present
		if avatarInterface, exists := peerRequest["avatar"]; exists {
			// Convert interface{} to AvatarData
			if avatarMap, ok := avatarInterface.(map[string]interface{}); ok {
				avatarData := &AvatarData{}
				if filename, ok := avatarMap["filename"].(string); ok {
					avatarData.Filename = filename
				}
				if data, ok := avatarMap["data"].(string); ok {
					avatarData.Data = data
				}
				if size, ok := avatarMap["size"].(float64); ok {
					avatarData.Size = int(size)
				}

				// Save the received avatar
				if err := p.saveReceivedAvatar(peerID, avatarData); err != nil {
					log.Printf("Failed to save avatar from peer %s: %v", peerID, err)
				}
			}
		}

		// Process friends data if present
		if friendsInterface, exists := peerRequest["friends"]; exists {
			if friendsSlice, ok := friendsInterface.([]interface{}); ok {
				friends := make([]models.Friend, 0, len(friendsSlice))
				for _, friendInterface := range friendsSlice {
					if friendMap, ok := friendInterface.(map[string]interface{}); ok {
						friend := models.Friend{}
						if peerID, ok := friendMap["peer_id"].(string); ok {
							friend.PeerID = peerID
						}
						if peerName, ok := friendMap["peer_name"].(string); ok {
							friend.PeerName = peerName
						}
						friends = append(friends, friend)
					}
				}
				
				// Save the received friends list
				if len(friends) > 0 {
					if err := p.dbService.SavePeerFriends(peerID.String(), friends); err != nil {
						log.Printf("Failed to save friends from peer %s: %v", peerID, err)
					} else {
						log.Printf("✅ Saved %d friends from peer %s", len(friends), peerID)
					}
				}
			}
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

	// Prepare our avatar data for transmission
	avatarData := p.prepareAvatarData()

	// Prepare our friends data for transmission
	friendsData := p.prepareFriendsData()

	ourRequest := map[string]interface{}{
		"app":     AppIdentifier,
		"version": "1.0.0",
		"nodeId":  p.host.ID().String(),
		"name":    ourNodeName,
	}

	// Include avatar data if available
	if avatarData != nil {
		ourRequest["avatar"] = avatarData
	}

	// Include friends data if available
	if friendsData != nil {
		ourRequest["friends"] = friendsData
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(ourRequest); err != nil {
		log.Printf("❌ Failed to send our identification to %s: %v", peerID, err)
		p.markPeerValidation(peerID, false)
		return false
	}

	// Read identification response from remote peer
	var response map[string]interface{}
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&response); err != nil {
		log.Printf("❌ Failed to decode identification from %s: %v", peerID, err)
		p.markPeerValidation(peerID, false)
		return false
	}

	// Check if it's our application
	app, exists := response["app"]
	if !exists {
		log.Printf("❌ Peer %s identification missing app field", peerID)
		p.markPeerValidation(peerID, false)
		return false
	}

	appStr, ok := app.(string)
	if !ok || appStr != AppIdentifier {
		log.Printf("❌ Peer %s is not running our application (app: %v)", peerID, app)
		p.markPeerValidation(peerID, false)
		return false
	}

	// Extract peer name if available
	peerName := "unknown"
	if name, exists := response["name"]; exists {
		if nameStr, ok := name.(string); ok {
			peerName = nameStr
		}
	}

	// Process avatar data if present
	if avatarInterface, exists := response["avatar"]; exists {
		// Convert interface{} to AvatarData
		if avatarMap, ok := avatarInterface.(map[string]interface{}); ok {
			receivedAvatarData := &AvatarData{}
			if filename, ok := avatarMap["filename"].(string); ok {
				receivedAvatarData.Filename = filename
			}
			if data, ok := avatarMap["data"].(string); ok {
				receivedAvatarData.Data = data
			}
			if size, ok := avatarMap["size"].(float64); ok {
				receivedAvatarData.Size = int(size)
			}

			// Save the received avatar
			if err := p.saveReceivedAvatar(peerID, receivedAvatarData); err != nil {
				log.Printf("Failed to save avatar from peer %s: %v", peerID, err)
			}
		}
	}

	// Process friends data if present
	if friendsInterface, exists := response["friends"]; exists {
		if friendsSlice, ok := friendsInterface.([]interface{}); ok {
			friends := make([]models.Friend, 0, len(friendsSlice))
			for _, friendInterface := range friendsSlice {
				if friendMap, ok := friendInterface.(map[string]interface{}); ok {
					friend := models.Friend{}
					if peerID, ok := friendMap["peer_id"].(string); ok {
						friend.PeerID = peerID
					}
					if peerName, ok := friendMap["peer_name"].(string); ok {
						friend.PeerName = peerName
					}
					friends = append(friends, friend)
				}
			}
			
			// Save the received friends list
			if len(friends) > 0 {
				if err := p.dbService.SavePeerFriends(peerID.String(), friends); err != nil {
					log.Printf("Failed to save friends from peer %s: %v", peerID, err)
				} else {
					log.Printf("✅ Saved %d friends from peer %s", len(friends), peerID)
				}
			}
		}
	}

	log.Printf("✅ Peer %s validated as our application (name: %s)", peerID, peerName)
	p.markPeerValidationWithName(peerID, true, peerName)
	return true
}

// markPeerValidation marks a peer as validated or not
func (p *P2PService) markPeerValidation(peerID peer.ID, isValid bool) {
	p.markPeerValidationWithName(peerID, isValid, "")
}

// checkPeerHasAvatar checks if a peer has avatar images available
func (p *P2PService) checkPeerHasAvatar(peerID peer.ID) bool {
	if p.container == nil || p.container.GetDirectoryService() == nil {
		return false
	}

	avatarImages, err := p.container.GetDirectoryService().GetPeerAvatarImages(peerID.String())
	if err != nil {
		return false
	}

	return len(avatarImages) > 0
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

		// Check if peer has avatar
		peerInfo.HasAvatar = p.checkPeerHasAvatar(peerID)

		// If peer is validated but missing name or avatar, schedule retry
		if isValid && (peerInfo.Name == "" || peerInfo.Name == "unknown" || !peerInfo.HasAvatar) {
			go p.retryPeerDataExchange(peerID)
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

// retryPeerDataExchange attempts to re-exchange identification data with a peer
func (p *P2PService) retryPeerDataExchange(peerID peer.ID) {
	// Wait a bit before retrying to avoid immediate retry storms
	time.Sleep(2 * time.Second)

	// Check if peer is still connected
	if p.host.Network().Connectedness(peerID) != network.Connected {
		return
	}

	log.Printf("🔄 Retrying data exchange with peer %s (missing name or avatar)", peerID)

	// Try to re-validate the peer to exchange identification data again
	if p.validatePeer(peerID) {
		log.Printf("✅ Successfully retried data exchange with peer %s", peerID)
	} else {
		log.Printf("❌ Failed to retry data exchange with peer %s", peerID)
	}
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
			Payload: p.getNodeInfo(),
		}

	case models.MessageTypeDiscovery:
		// Handle discovery request
		response = models.P2PMessage{
			Type:    models.MessageTypeDiscoveryResp,
			Payload: p.getNodeInfo(),
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

	case models.MessageTypeGetDocs:
		// Handle docs list request
		log.Printf("📝 Processing docs request from %s", peerID)
		docsResponse := p.handleGetDocsRequest()
		response = models.P2PMessage{
			Type:    models.MessageTypeGetDocsResp,
			Payload: docsResponse,
		}

	case models.MessageTypeGetDoc:
		// Handle specific doc request
		log.Printf("📝 Processing doc request from %s", peerID)
		docResponse := p.handleGetDocRequest(msg.Payload)
		response = models.P2PMessage{
			Type:    models.MessageTypeGetDocResp,
			Payload: docResponse,
		}

	case models.MessageTypeGetFiles:
		// Handle files table request
		log.Printf("📁 Processing files table request from %s", peerID)
		filesResponse := p.handleGetFilesRequest()
		response = models.P2PMessage{
			Type:    models.MessageTypeGetFilesResp,
			Payload: filesResponse,
		}

	case models.MessageTypeGetGalleries:
		// Handle galleries request
		log.Printf("📷 Processing galleries request from %s", peerID)
		galleriesResponse := p.handleGetGalleriesRequest()
		response = models.P2PMessage{
			Type:    models.MessageTypeGetGalleriesResp,
			Payload: galleriesResponse,
		}

	case models.MessageTypeGetGallery:
		// Handle specific gallery request
		log.Printf("📷 Processing gallery request from %s", peerID)
		galleryResponse := p.handleGetGalleryRequest(msg.Payload)
		response = models.P2PMessage{
			Type:    models.MessageTypeGetGalleryResp,
			Payload: galleryResponse,
		}

	case models.MessageTypeGetGalleryImage:
		// Handle gallery image request
		log.Printf("📷 Processing gallery image request from %s", peerID)
		imageResponse := p.handleGetGalleryImageRequest(msg.Payload)
		response = models.P2PMessage{
			Type:    models.MessageTypeGetGalleryImageResp,
			Payload: imageResponse,
		}

	case models.MessageTypeGetFriends:
		// Handle friends list request
		log.Printf("👥 Processing friends request from %s", peerID)
		friendsResponse := p.handleGetFriendsRequest()
		response = models.P2PMessage{
			Type:    models.MessageTypeGetFriendsResp,
			Payload: friendsResponse,
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

// startPeerDataRetry runs periodic checks for missing peer data and retries
func (p *P2PService) startPeerDataRetry() {
	ticker := time.NewTicker(60 * time.Second) // Check every 60 seconds
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.checkAndRetryMissingPeerData()
		}
	}
}

// checkAndRetryMissingPeerData checks for peers with missing names or avatars and retries
func (p *P2PService) checkAndRetryMissingPeerData() {
	p.peerInfoMutex.RLock()
	var peersToRetry []peer.ID

	for peerID, peerInfo := range p.connectedPeers {
		// Skip if peer is not validated or not connected
		if !peerInfo.IsValidated || p.host.Network().Connectedness(peerID) != network.Connected {
			continue
		}

		// Check if peer has missing name or avatar
		if (peerInfo.Name == "" || peerInfo.Name == "unknown") || !peerInfo.HasAvatar {
			peersToRetry = append(peersToRetry, peerID)
		}
	}
	p.peerInfoMutex.RUnlock()

	if len(peersToRetry) > 0 {
		log.Printf("🔄 Found %d peers with missing data, scheduling retries", len(peersToRetry))

		// Retry data exchange for each peer
		for _, peerID := range peersToRetry {
			go p.retryPeerDataExchange(peerID)
		}
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

		// Update address if we have a new one (handle case where peer reconnects from different address)
		if address != "" {
			peerInfo.Addresses = []string{address}
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

// handleGetDocsRequest handles P2P request for docs list
func (p *P2PService) handleGetDocsRequest() *models.DocsResponse {
	if p.container == nil || p.container.GetDirectoryService() == nil {
		return &models.DocsResponse{
			Docs:  []models.Doc{},
			Count: 0,
		}
	}

	docs, err := p.container.GetDirectoryService().GetDocs()
	if err != nil {
		log.Printf("Failed to get docs for P2P request: %v", err)
		return &models.DocsResponse{
			Docs:  []models.Doc{},
			Count: 0,
		}
	}

	return &models.DocsResponse{
		Docs:  docs,
		Count: len(docs),
	}
}

// handleGetFriendsRequest handles P2P request for friends list
func (p *P2PService) handleGetFriendsRequest() *models.FriendsResponse {
	if p.dbService == nil {
		return &models.FriendsResponse{
			Friends: []models.Friend{},
			Count:   0,
		}
	}

	friends, err := p.dbService.GetFriends()
	if err != nil {
		log.Printf("Failed to get friends for P2P request: %v", err)
		return &models.FriendsResponse{
			Friends: []models.Friend{},
			Count:   0,
		}
	}

	return &models.FriendsResponse{
		Friends: friends,
		Count:   len(friends),
	}
}

// handleGetDocRequest handles P2P request for specific doc
func (p *P2PService) handleGetDocRequest(payload interface{}) *models.DocResponse {
	if p.container == nil || p.container.GetDirectoryService() == nil {
		return &models.DocResponse{
			Doc: nil,
		}
	}

	// Parse the request payload
	requestData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal doc request payload: %v", err)
		return &models.DocResponse{Doc: nil}
	}

	var docRequest models.DocRequest
	if err := json.Unmarshal(requestData, &docRequest); err != nil {
		log.Printf("Failed to parse doc request: %v", err)
		return &models.DocResponse{Doc: nil}
	}

	doc, err := p.container.GetDirectoryService().GetDoc(docRequest.Filename)
	if err != nil {
		log.Printf("Failed to get doc %s for P2P request: %v", docRequest.Filename, err)
		return &models.DocResponse{Doc: nil}
	}

	return &models.DocResponse{Doc: doc}
}

// handleGetFilesRequest handles P2P request for files table
func (p *P2PService) handleGetFilesRequest() *models.FilesResponse {
	if p.container == nil || p.container.GetDatabase() == nil {
		return &models.FilesResponse{
			Files:  []models.FileRecord{},
			PeerID: p.GetNode().ID.String(),
			Count:  0,
		}
	}

	files, err := p.container.GetDatabase().GetFiles()
	if err != nil {
		log.Printf("Failed to get files for P2P request: %v", err)
		return &models.FilesResponse{
			Files:  []models.FileRecord{},
			PeerID: p.GetNode().ID.String(),
			Count:  0,
		}
	}

	// Filter to only return files owned by this peer
	var ownFiles []models.FileRecord
	myPeerID := p.GetNode().ID.String()
	for _, file := range files {
		if file.PeerID == myPeerID {
			ownFiles = append(ownFiles, file)
		}
	}

	return &models.FilesResponse{
		Files:  ownFiles,
		PeerID: myPeerID,
		Count:  len(ownFiles),
	}
}

// RequestPeerDocs requests docs list from a peer
func (p *P2PService) RequestPeerDocs(peerID string) (*models.DocsResponse, error) {
	peer, err := peer.Decode(peerID)
	if err != nil {
		return nil, fmt.Errorf("invalid peer ID: %w", err)
	}

	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer cancel()

	stream, err := p.host.NewStream(ctx, peer, protocol.ID(AppProtocol))
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}
	defer stream.Close()

	// Send docs request
	msg := models.P2PMessage{
		Type:    models.MessageTypeGetDocs,
		Payload: models.DocsRequest{},
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(msg); err != nil {
		return nil, fmt.Errorf("failed to send docs request: %w", err)
	}

	// Read response
	decoder := json.NewDecoder(stream)
	var response models.P2PMessage
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Type != models.MessageTypeGetDocsResp {
		return nil, fmt.Errorf("unexpected response type: %s", response.Type)
	}

	// Parse response payload
	responseData, err := json.Marshal(response.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response payload: %w", err)
	}

	var docsResponse models.DocsResponse
	if err := json.Unmarshal(responseData, &docsResponse); err != nil {
		return nil, fmt.Errorf("failed to parse docs response: %w", err)
	}

	return &docsResponse, nil
}

// RequestPeerFiles requests files table from a peer
func (p *P2PService) RequestPeerFiles(peerID string) (*models.FilesResponse, error) {
	peer, err := peer.Decode(peerID)
	if err != nil {
		return nil, fmt.Errorf("invalid peer ID: %w", err)
	}

	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer cancel()

	stream, err := p.host.NewStream(ctx, peer, protocol.ID(AppProtocol))
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}
	defer stream.Close()

	// Send files request
	msg := models.P2PMessage{
		Type:    models.MessageTypeGetFiles,
		Payload: models.FilesRequest{},
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(msg); err != nil {
		return nil, fmt.Errorf("failed to send files request: %w", err)
	}

	// Read response
	decoder := json.NewDecoder(stream)
	var response models.P2PMessage
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Type != models.MessageTypeGetFilesResp {
		return nil, fmt.Errorf("unexpected response type: %s", response.Type)
	}

	// Parse response payload
	responseData, err := json.Marshal(response.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response payload: %w", err)
	}

	var filesResponse models.FilesResponse
	if err := json.Unmarshal(responseData, &filesResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal files response: %w", err)
	}

	return &filesResponse, nil
}

// RequestPeerDoc requests a specific doc from a peer
func (p *P2PService) RequestPeerDoc(peerID, filename string) (*models.DocResponse, error) {
	peer, err := peer.Decode(peerID)
	if err != nil {
		return nil, fmt.Errorf("invalid peer ID: %w", err)
	}

	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer cancel()

	stream, err := p.host.NewStream(ctx, peer, protocol.ID(AppProtocol))
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}
	defer stream.Close()

	// Send doc request
	msg := models.P2PMessage{
		Type: models.MessageTypeGetDoc,
		Payload: models.DocRequest{
			Filename: filename,
		},
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(msg); err != nil {
		return nil, fmt.Errorf("failed to send doc request: %w", err)
	}

	// Read response
	decoder := json.NewDecoder(stream)
	var response models.P2PMessage
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Type != models.MessageTypeGetDocResp {
		return nil, fmt.Errorf("unexpected response type: %s", response.Type)
	}

	// Parse response payload
	responseData, err := json.Marshal(response.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response payload: %w", err)
	}

	var docResponse models.DocResponse
	if err := json.Unmarshal(responseData, &docResponse); err != nil {
		return nil, fmt.Errorf("failed to parse doc response: %w", err)
	}

	return &docResponse, nil
}

// handleGetGalleriesRequest handles P2P request for galleries list
func (p *P2PService) handleGetGalleriesRequest() *models.GalleriesResponse {
	if p.container == nil || p.container.GetDirectoryService() == nil {
		return &models.GalleriesResponse{
			Galleries: []models.MediaGallery{},
			Count:     0,
		}
	}

	galleries, err := p.container.GetDirectoryService().GetMediaGalleries(models.MediaTypeImage)
	if err != nil {
		log.Printf("Failed to get galleries for P2P request: %v", err)
		return &models.GalleriesResponse{
			Galleries: []models.MediaGallery{},
			Count:     0,
		}
	}

	return &models.GalleriesResponse{
		Galleries: galleries,
		Count:     len(galleries),
	}
}

// handleGetGalleryRequest handles P2P request for specific gallery
func (p *P2PService) handleGetGalleryRequest(payload interface{}) *models.GalleryResponse {
	if p.container == nil || p.container.GetDirectoryService() == nil {
		return &models.GalleryResponse{
			Gallery: nil,
		}
	}

	// Parse the request payload
	requestData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal gallery request payload: %v", err)
		return &models.GalleryResponse{Gallery: nil}
	}

	var galleryRequest models.GalleryRequest
	if err := json.Unmarshal(requestData, &galleryRequest); err != nil {
		log.Printf("Failed to parse gallery request: %v", err)
		return &models.GalleryResponse{Gallery: nil}
	}

	// Get gallery files using unified method
	files, err := p.container.GetDirectoryService().GetMediaGalleryFiles(models.MediaTypeImage, galleryRequest.GalleryName)
	if err != nil {
		log.Printf("Failed to get gallery %s for P2P request: %v", galleryRequest.GalleryName, err)
		return &models.GalleryResponse{Gallery: nil}
	}

	gallery := &models.MediaGallery{
		Name:      galleryRequest.GalleryName,
		MediaType: models.MediaTypeImage,
		FileCount: len(files),
		Files:     files,
	}

	return &models.GalleryResponse{Gallery: gallery}
}

// handleGetGalleryImageRequest handles P2P request for specific gallery image
func (p *P2PService) handleGetGalleryImageRequest(payload interface{}) *models.GalleryImageResponse {
	if p.container == nil || p.container.GetDirectoryService() == nil {
		return &models.GalleryImageResponse{
			ImageData: "",
			Filename:  "",
			Size:      0,
		}
	}

	// Parse the request payload
	requestData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal gallery image request payload: %v", err)
		return &models.GalleryImageResponse{ImageData: "", Filename: "", Size: 0}
	}

	var imageRequest models.GalleryImageRequest
	if err := json.Unmarshal(requestData, &imageRequest); err != nil {
		log.Printf("Failed to parse gallery image request: %v", err)
		return &models.GalleryImageResponse{ImageData: "", Filename: "", Size: 0}
	}

	// Read the image file
	imagesDir := p.container.GetDirectoryService().GetDirectoryPath()
	imagePath := filepath.Join(imagesDir, "images", imageRequest.GalleryName, imageRequest.ImageName)

	// Validate that the file exists and is within the gallery directory using unified method
	galleryImages, err := p.container.GetDirectoryService().GetMediaGalleryFiles(models.MediaTypeImage, imageRequest.GalleryName)
	if err != nil {
		log.Printf("Failed to get gallery images for validation: %v", err)
		return &models.GalleryImageResponse{ImageData: "", Filename: "", Size: 0}
	}

	// Check if the requested image exists in the gallery
	found := false
	for _, img := range galleryImages {
		if img == imageRequest.ImageName {
			found = true
			break
		}
	}

	if !found {
		log.Printf("Image %s not found in gallery %s", imageRequest.ImageName, imageRequest.GalleryName)
		return &models.GalleryImageResponse{ImageData: "", Filename: "", Size: 0}
	}

	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		log.Printf("Failed to read image file %s: %v", imagePath, err)
		return &models.GalleryImageResponse{ImageData: "", Filename: "", Size: 0}
	}

	// Limit image size to 5MB for transmission
	if len(imageData) > 5*1024*1024 {
		log.Printf("Image %s too large (%d bytes), skipping", imageRequest.ImageName, len(imageData))
		return &models.GalleryImageResponse{ImageData: "", Filename: "", Size: 0}
	}

	// Encode to base64
	encodedData := base64.StdEncoding.EncodeToString(imageData)

	return &models.GalleryImageResponse{
		ImageData: encodedData,
		Filename:  imageRequest.ImageName,
		Size:      len(imageData),
	}
}

// RequestPeerGalleries requests galleries list from a peer
func (p *P2PService) RequestPeerGalleries(peerID string) (*models.GalleriesResponse, error) {
	peer, err := peer.Decode(peerID)
	if err != nil {
		return nil, fmt.Errorf("invalid peer ID: %w", err)
	}

	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer cancel()

	stream, err := p.host.NewStream(ctx, peer, protocol.ID(AppProtocol))
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}
	defer stream.Close()

	// Send galleries request
	msg := models.P2PMessage{
		Type:    models.MessageTypeGetGalleries,
		Payload: models.GalleriesRequest{},
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(msg); err != nil {
		return nil, fmt.Errorf("failed to send galleries request: %w", err)
	}

	// Read response
	decoder := json.NewDecoder(stream)
	var response models.P2PMessage
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Type != models.MessageTypeGetGalleriesResp {
		return nil, fmt.Errorf("unexpected response type: %s", response.Type)
	}

	// Parse response payload
	responseData, err := json.Marshal(response.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response payload: %w", err)
	}

	var galleriesResponse models.GalleriesResponse
	if err := json.Unmarshal(responseData, &galleriesResponse); err != nil {
		return nil, fmt.Errorf("failed to parse galleries response: %w", err)
	}

	return &galleriesResponse, nil
}

// RequestPeerGallery requests a specific gallery from a peer
func (p *P2PService) RequestPeerGallery(peerID, galleryName string) (*models.GalleryResponse, error) {
	peer, err := peer.Decode(peerID)
	if err != nil {
		return nil, fmt.Errorf("invalid peer ID: %w", err)
	}

	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer cancel()

	stream, err := p.host.NewStream(ctx, peer, protocol.ID(AppProtocol))
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}
	defer stream.Close()

	// Send gallery request
	msg := models.P2PMessage{
		Type: models.MessageTypeGetGallery,
		Payload: models.GalleryRequest{
			GalleryName: galleryName,
		},
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(msg); err != nil {
		return nil, fmt.Errorf("failed to send gallery request: %w", err)
	}

	// Read response
	decoder := json.NewDecoder(stream)
	var response models.P2PMessage
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Type != models.MessageTypeGetGalleryResp {
		return nil, fmt.Errorf("unexpected response type: %s", response.Type)
	}

	// Parse response payload
	responseData, err := json.Marshal(response.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response payload: %w", err)
	}

	var galleryResponse models.GalleryResponse
	if err := json.Unmarshal(responseData, &galleryResponse); err != nil {
		return nil, fmt.Errorf("failed to parse gallery response: %w", err)
	}

	return &galleryResponse, nil
}

// RequestPeerGalleryImage requests a specific image from a peer's gallery
func (p *P2PService) RequestPeerGalleryImage(peerID, galleryName, imageName string) (*models.GalleryImageResponse, error) {
	peer, err := peer.Decode(peerID)
	if err != nil {
		return nil, fmt.Errorf("invalid peer ID: %w", err)
	}

	ctx, cancel := context.WithTimeout(p.ctx, 30*time.Second) // Longer timeout for image transfer
	defer cancel()

	stream, err := p.host.NewStream(ctx, peer, protocol.ID(AppProtocol))
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}
	defer stream.Close()

	// Send gallery image request
	msg := models.P2PMessage{
		Type: models.MessageTypeGetGalleryImage,
		Payload: models.GalleryImageRequest{
			GalleryName: galleryName,
			ImageName:   imageName,
		},
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(msg); err != nil {
		return nil, fmt.Errorf("failed to send gallery image request: %w", err)
	}

	// Read response
	decoder := json.NewDecoder(stream)
	var response models.P2PMessage
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Type != models.MessageTypeGetGalleryImageResp {
		return nil, fmt.Errorf("unexpected response type: %s", response.Type)
	}

	// Parse response payload
	responseData, err := json.Marshal(response.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response payload: %w", err)
	}

	var imageResponse models.GalleryImageResponse
	if err := json.Unmarshal(responseData, &imageResponse); err != nil {
		return nil, fmt.Errorf("failed to parse gallery image response: %w", err)
	}

	return &imageResponse, nil
}

// getNodeInfo creates a NodeInfoResponse for P2P communication
func (p *P2PService) getNodeInfo() *models.NodeInfoResponse {
	response := &models.NodeInfoResponse{
		Node:         p.GetNode(),
		IsPublicNode: p.IsPublicNode(),
	}

	// Add peer information if available
	peerInfo := p.GetConnectedPeerInfo()
	if len(peerInfo) > 0 {
		response.ConnectedPeerInfo = make(map[string]*models.PeerInfoJSON)
		for peerID, info := range peerInfo {
			response.ConnectedPeerInfo[peerID.String()] = &models.PeerInfoJSON{
				ID:             info.ID.String(),
				Addresses:      info.Addresses,
				FirstSeen:      info.FirstSeen,
				LastSeen:       info.LastSeen,
				IsValidated:    info.IsValidated,
				ConnectionType: info.ConnectionType,
				Name:           info.Name,
				HasAvatar:      info.HasAvatar,
			}
		}
	}

	return response
}

// FetchPeerFriends requests friends list from a remote peer
func (p *P2PService) FetchPeerFriends(peerID peer.ID) ([]models.Friend, error) {
	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer cancel()

	stream, err := p.host.NewStream(ctx, peerID, protocol.ID(AppProtocol))
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}
	defer stream.Close()

	// Send friends request
	msg := models.P2PMessage{
		Type:    models.MessageTypeGetFriends,
		Payload: models.FriendsRequest{},
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(msg); err != nil {
		return nil, fmt.Errorf("failed to send friends request: %w", err)
	}

	// Read response
	decoder := json.NewDecoder(stream)
	var response models.P2PMessage
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Type != models.MessageTypeGetFriendsResp {
		return nil, fmt.Errorf("unexpected response type: %s", response.Type)
	}

	// Parse response payload
	responseData, err := json.Marshal(response.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response payload: %w", err)
	}

	var friendsResponse models.FriendsResponse
	if err := json.Unmarshal(responseData, &friendsResponse); err != nil {
		return nil, fmt.Errorf("failed to parse friends response: %w", err)
	}

	log.Printf("✅ Successfully fetched %d friends from peer %s", len(friendsResponse.Friends), peerID)
	return friendsResponse.Friends, nil
}

// FetchAndSavePeerFriends fetches friends from a remote peer and saves them to the database
func (p *P2PService) FetchAndSavePeerFriends(peerIDStr string) ([]models.Friend, error) {
	// First, check if we have connection info for this peer
	connectionHistory, err := p.dbService.GetConnectionHistory()
	if err != nil {
		return nil, fmt.Errorf("failed to get connection history: %w", err)
	}

	var targetConnection *models.ConnectionRecord
	for _, conn := range connectionHistory {
		if conn.PeerID == peerIDStr {
			targetConnection = &conn
			break
		}
	}

	if targetConnection == nil {
		return nil, fmt.Errorf("no connection information found for peer %s", peerIDStr)
	}

	// Parse the peer ID
	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid peer ID: %w", err)
	}

	// Fetch friends from the remote peer
	friends, err := p.FetchPeerFriends(peerID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch friends from peer: %w", err)
	}

	// Save the fetched friends to the peer_friends table
	if len(friends) > 0 {
		if err := p.dbService.SavePeerFriends(peerIDStr, friends); err != nil {
			return nil, fmt.Errorf("failed to save peer friends: %w", err)
		}

		// Save fetched friends as connection records for potential future connections
		for _, friend := range friends {
			// Check if we already have a connection record for this friend
			connectionExists := false
			for _, conn := range connectionHistory {
				if conn.PeerID == friend.PeerID {
					connectionExists = true
					break
				}
			}

			// If no connection record exists, create a placeholder connection record
			if !connectionExists {
				// Use a placeholder address since we don't know the actual address
				err := p.dbService.RecordConnectionWithName(
					friend.PeerID,
					"unknown:0",   // placeholder address
					"peer_friend", // connection type to indicate this was discovered through friends
					false,         // not validated yet
					friend.PeerName,
				)
				if err != nil {
					log.Printf("⚠️ Failed to record connection for friend %s: %v", friend.PeerID, err)
				} else {
					log.Printf("📝 Recorded placeholder connection for friend %s (%s)", friend.PeerName, friend.PeerID)
				}
			}
		}
	}

	return friends, nil
}

// Close shuts down the P2P service
func (p *P2PService) Close() error {
	p.cancel()
	if p.dht != nil {
		p.dht.Close()
	}
	return p.host.Close()
}
