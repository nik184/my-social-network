package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	ServiceTag = "my-social-network"
)

// P2PService handles libp2p networking
type P2PService struct {
	host      host.Host
	dht       *dht.IpfsDHT
	ctx       context.Context
	cancel    context.CancelFunc
	appService *AppService
}

// NewP2PService creates a new P2P service
func NewP2PService(appService *AppService) (*P2PService, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
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
	
	// Create libp2p host with available ports and NAT traversal
	h, err := libp2p.New(
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", tcpPort),        // TCP on available port
			fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", quicPort),  // QUIC on available port
		),
		libp2p.ConnectionManager(connmgr),
		libp2p.EnableHolePunching(),      // Enable hole punching
		libp2p.EnableNATService(),        // Enable NAT service
		libp2p.DefaultSecurity,           // Use default security protocols
		libp2p.DefaultMuxers,             // Use default stream multiplexers
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
		host:       h,
		ctx:        ctx,
		cancel:     cancel,
		appService: appService,
	}
	
	// Set stream handler for our protocol
	h.SetStreamHandler(protocol.ID(AppProtocol), service.handleStream)
	
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
	
	// Connect to bootstrap nodes
	go p.connectToBootstrapNodes()
	
	// Setup relay discovery after DHT is ready
	go p.setupRelayDiscovery()
	
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

// connectToBootstrapNodes connects to well-known bootstrap nodes
func (p *P2PService) connectToBootstrapNodes() {
	// IPFS bootstrap nodes for initial connectivity
	bootstrapNodes := []string{
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
	}
	
	var wg sync.WaitGroup
	for _, nodeAddr := range bootstrapNodes {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			
			maddr, err := multiaddr.NewMultiaddr(addr)
			if err != nil {
				log.Printf("Invalid bootstrap address %s: %v", addr, err)
				return
			}
			
			peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
			if err != nil {
				log.Printf("Failed to get peer info from %s: %v", addr, err)
				return
			}
			
			ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
			defer cancel()
			
			if err := p.host.Connect(ctx, *peerInfo); err != nil {
				log.Printf("Failed to connect to bootstrap node %s: %v", addr, err)
			} else {
				log.Printf("Connected to bootstrap node: %s", peerInfo.ID)
			}
		}(nodeAddr)
	}
	wg.Wait()
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
	log.Printf("Discovered peer via mDNS: %s", peerInfo.ID)
	
	// Connect to discovered peer
	ctx, cancel := context.WithTimeout(n.p2pService.ctx, 5*time.Second)
	defer cancel()
	
	if err := n.p2pService.host.Connect(ctx, peerInfo); err != nil {
		log.Printf("Failed to connect to discovered peer %s: %v", peerInfo.ID, err)
	} else {
		log.Printf("Connected to peer: %s", peerInfo.ID)
	}
}

// handleStream handles incoming streams
func (p *P2PService) handleStream(stream network.Stream) {
	defer stream.Close()
	
	var msg models.P2PMessage
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&msg); err != nil {
		log.Printf("Failed to decode message: %v", err)
		return
	}
	
	log.Printf("Received message type: %s from %s", msg.Type, stream.Conn().RemotePeer())
	
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

// GetConnectedPeers returns list of connected peers
func (p *P2PService) GetConnectedPeers() []peer.ID {
	return p.host.Network().Peers()
}

// Close shuts down the P2P service
func (p *P2PService) Close() error {
	p.cancel()
	if p.dht != nil {
		p.dht.Close()
	}
	return p.host.Close()
}