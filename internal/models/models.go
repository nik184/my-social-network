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

// Note represents a text note from the notes directory
type Note struct {
	Filename    string    `json:"filename"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Preview     string    `json:"preview"`
	ModifiedAt  time.Time `json:"modified_at"`
	Size        int64     `json:"size"`
}

// Gallery represents a photo gallery (subdirectory in images/)
type Gallery struct {
	Name       string   `json:"name"`
	ImageCount int      `json:"image_count"`
	Images     []string `json:"images"`
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
	Name           string    `json:"name"`
	HasAvatar      bool      `json:"has_avatar"`
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

// SecondDegreePeer represents a peer that is connected to one of our direct connections
type SecondDegreePeer struct {
	PeerID           string `json:"peer_id"`
	PeerName         string `json:"peer_name"`
	ViaPeerID        string `json:"via_peer_id"`
	ViaPeerName      string `json:"via_peer_name"`
	ConnectionPath   string `json:"connection_path,omitempty"`
}

// SecondDegreeConnectionsResponse represents the response for second-degree peer discovery
type SecondDegreeConnectionsResponse struct {
	Peers []SecondDegreePeer `json:"peers"`
	Count int                `json:"count"`
}

// SecondDegreeConnectionRequest represents a request to connect to a second-degree peer
type SecondDegreeConnectionRequest struct {
	TargetPeerID string `json:"targetPeerId"`
	ViaPeerID    string `json:"viaPeerId"`
}

// PeerListItem represents a peer in a peer list response
type PeerListItem struct {
	PeerID   string `json:"peer_id"`
	PeerName string `json:"peer_name"`
}

// PeerListResponse represents a response containing a list of connected peers
type PeerListResponse struct {
	Peers []PeerListItem `json:"peers"`
	Count int            `json:"count"`
}

// ConnectionHistoryItem represents a connection history item with current status
type ConnectionHistoryItem struct {
	PeerID             string    `json:"peer_id"`
	PeerName           string    `json:"peer_name"`
	Address            string    `json:"address"`
	LastConnected      time.Time `json:"last_connected"`
	ConnectionType     string    `json:"connection_type"`
	IsValidated        bool      `json:"is_validated"`
	CurrentlyConnected bool      `json:"currently_connected"`
}

// ConnectionHistoryResponse represents the response for connection history
type ConnectionHistoryResponse struct {
	Connections []ConnectionHistoryItem `json:"connections"`
	Count       int                     `json:"count"`
}

// Friend represents a friend in the friends list
type Friend struct {
	ID       int       `json:"id"`
	PeerID   string    `json:"peer_id"`
	PeerName string    `json:"peer_name"`
	AddedAt  time.Time `json:"added_at"`
	LastSeen *time.Time `json:"last_seen"`
	IsOnline bool      `json:"is_online"`
}

// FriendsResponse represents the response for friends list
type FriendsResponse struct {
	Friends []Friend `json:"friends"`
	Count   int      `json:"count"`
}

// AddFriendRequest represents a request to add a friend
type AddFriendRequest struct {
	PeerID   string `json:"peer_id"`
	PeerName string `json:"peer_name"`
}

// NotesRequest represents a P2P request for notes list
type NotesRequest struct {
	// Currently no additional fields needed
}

// NotesResponse represents a P2P response with notes list
type NotesResponse struct {
	Notes []Note `json:"notes"`
	Count int    `json:"count"`
}

// NoteRequest represents a P2P request for a specific note
type NoteRequest struct {
	Filename string `json:"filename"`
}

// NoteResponse represents a P2P response with note content
type NoteResponse struct {
	Note *Note `json:"note"`
}

// Constants for message types
const (
	MessageTypeGetInfo          = "getInfo"
	MessageTypeGetInfoResp      = "getInfoResp"
	MessageTypeDiscovery        = "discovery"
	MessageTypeDiscoveryResp    = "discoveryResp"
	MessageTypeGetPeerList      = "getPeerList"
	MessageTypeGetPeerListResp  = "getPeerListResp"
	MessageTypeHolePunchAssist  = "holePunchAssist"
	MessageTypeHolePunchResp    = "holePunchResp"
	MessageTypeGetNotes         = "getNotes"
	MessageTypeGetNotesResp     = "getNotesResp"
	MessageTypeGetNote          = "getNote"
	MessageTypeGetNoteResp      = "getNoteResp"
)