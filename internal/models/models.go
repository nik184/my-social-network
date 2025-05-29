package models

import (
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// FolderInfo represents information about a scanned directory
type FolderInfo struct {
	Path     string    `json:"path"`
	Files    []string  `json:"files"`
	LastScan time.Time `json:"lastScan"`
}

// NetworkNode represents a node in the distributed network
type NetworkNode struct {
	ID        peer.ID             `json:"id"`
	Addresses []multiaddr.Multiaddr `json:"addresses"`
	LastSeen  time.Time           `json:"lastSeen"`
}

// DiscoveryRequest represents a request to discover a node
type DiscoveryRequest struct {
	PeerID string `json:"peerId"`
}

// IPConnectionRequest represents a request to connect to a node by IP address
type IPConnectionRequest struct {
	IP     string `json:"ip"`
	Port   int    `json:"port"`
	PeerID string `json:"peerId"`
}

// ConnectionInfo represents the connection information for sharing
type ConnectionInfo struct {
	PeerID         string   `json:"peerId"`
	PublicAddress  string   `json:"publicAddress,omitempty"`
	Port           int      `json:"port,omitempty"`
	LocalAddresses []string `json:"localAddresses"`
	IsPublicNode   bool     `json:"isPublicNode"`
}

// PeerInfo stores information about connected peers for JSON serialization
type PeerInfoJSON struct {
	ID             string    `json:"id"`
	Addresses      []string  `json:"addresses"`
	FirstSeen      time.Time `json:"first_seen"`
	LastSeen       time.Time `json:"last_seen"`
	IsValidated    bool      `json:"is_validated"`
	ConnectionType string    `json:"connection_type"`
}

// NodeInfoResponse represents the response containing node and folder information
type NodeInfoResponse struct {
	FolderInfo        *FolderInfo                   `json:"folderInfo"`
	Node              *NetworkNode                  `json:"node"`
	IsPublicNode      bool                          `json:"isPublicNode"`
	ConnectedPeerInfo map[string]*PeerInfoJSON      `json:"connectedPeerInfo,omitempty"`
}

// StatusResponse represents a generic status response
type StatusResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// P2PMessage represents a message sent over libp2p streams
type P2PMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Constants for message types
const (
	MessageTypeGetInfo      = "getInfo"
	MessageTypeGetInfoResp  = "getInfoResp"
	MessageTypeDiscovery    = "discovery"
	MessageTypeDiscoveryResp = "discoveryResp"
)