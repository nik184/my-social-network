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

// NodeInfoResponse represents the response containing node and folder information
type NodeInfoResponse struct {
	FolderInfo *FolderInfo  `json:"folderInfo"`
	Node       *NetworkNode `json:"node"`
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