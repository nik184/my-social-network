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

// Doc represents a text doc from the docs directory
type Doc struct {
	Filename    string    `json:"filename"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Preview     string    `json:"preview"`
	ModifiedAt  time.Time `json:"modified_at"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"` // "text" for .txt files, "html" for .md files (converted to HTML)
}

// MediaType represents the type of media
type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeAudio MediaType = "audio"
	MediaTypeVideo MediaType = "video"
)

// MediaGallery represents a unified gallery for any media type
type MediaGallery struct {
	Name      string    `json:"name"`
	MediaType MediaType `json:"media_type"`
	FileCount int       `json:"file_count"`
	Files     []string  `json:"files"`
}


// ConnectionRecord represents a connection history record
type ConnectionRecord struct {
	ID             int       `json:"id"`
	PeerID         string    `json:"peer_id"`
	Address        string    `json:"address"`
	FirstConnected time.Time `json:"first_connected"`
	LastConnected  time.Time `json:"last_connected"`
	ConnectionType string    `json:"connection_type"`
	IsValidated    bool      `json:"is_validated"`
	PeerName       string    `json:"peer_name"`
}

// FileRecord represents a file metadata record
type FileRecord struct {
	ID        int       `json:"id"`
	FilePath  string    `json:"filepath"`
	Hash      string    `json:"hash"`
	Size      int64     `json:"size"`
	Extension string    `json:"extension"`
	Type      string    `json:"type"`
	PeerID    string    `json:"peer_id"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NetworkNode represents a node in the distributed network
type NetworkNode struct {
	ID        peer.ID               `json:"id"`
	Addresses []multiaddr.Multiaddr `json:"addresses"`
	LastSeen  time.Time             `json:"lastSeen"`
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
	FolderInfo        *FolderInfo              `json:"folderInfo"`
	Node              *NetworkNode             `json:"node"`
	IsPublicNode      bool                     `json:"isPublicNode"`
	ConnectedPeerInfo map[string]*PeerInfoJSON `json:"connectedPeerInfo,omitempty"`
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

// Friend represents a friend in the friends list
type Friend struct {
	ID       int        `json:"id"`
	PeerID   string     `json:"peer_id"`
	PeerName string     `json:"peer_name"`
	AddedAt  time.Time  `json:"added_at"`
	LastSeen *time.Time `json:"last_seen"`
	IsOnline bool       `json:"is_online"`
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

// FilesRequest represents a request for files table
type FilesRequest struct {
	RequestID string `json:"request_id"`
}

// FilesResponse represents a response with files table data
type FilesResponse struct {
	RequestID string       `json:"request_id"`
	Files     []FileRecord `json:"files"`
	PeerID    string       `json:"peer_id"`
	Count     int          `json:"count"`
}

// DocsRequest represents a P2P request for docs list
type DocsRequest struct {
	// Currently no additional fields needed
}

// DocsResponse represents a P2P response with docs list
type DocsResponse struct {
	Docs  []Doc `json:"docs"`
	Count int   `json:"count"`
}

// DocRequest represents a P2P request for a specific doc
type DocRequest struct {
	Filename string `json:"filename"`
}

// DocResponse represents a P2P response with doc content
type DocResponse struct {
	Doc *Doc `json:"doc"`
}

// MediaGalleriesRequest represents a P2P request for media galleries list
type MediaGalleriesRequest struct {
	MediaType MediaType `json:"media_type"`
}

// MediaGalleriesResponse represents a P2P response with media galleries list
type MediaGalleriesResponse struct {
	MediaType MediaType      `json:"media_type"`
	Galleries []MediaGallery `json:"galleries"`
	Count     int            `json:"count"`
}

// MediaGalleryRequest represents a P2P request for a specific media gallery
type MediaGalleryRequest struct {
	MediaType   MediaType `json:"media_type"`
	GalleryName string    `json:"gallery_name"`
}

// MediaGalleryResponse represents a P2P response with media gallery content
type MediaGalleryResponse struct {
	Gallery *MediaGallery `json:"gallery"`
}

// MediaFileRequest represents a P2P request for a specific media file
type MediaFileRequest struct {
	MediaType   MediaType `json:"media_type"`
	GalleryName string    `json:"gallery_name"`
	FileName    string    `json:"file_name"`
}

// MediaFileResponse represents a P2P response with media file data
type MediaFileResponse struct {
	MediaType MediaType `json:"media_type"`
	FileData  string    `json:"file_data"` // base64 encoded file data
	Filename  string    `json:"filename"`
	Size      int       `json:"size"`
}

// Legacy compatibility types for P2P communication
type GalleriesResponse struct {
	Galleries []MediaGallery `json:"galleries"`
	Count     int            `json:"count"`
}

type GalleryResponse struct {
	Gallery *MediaGallery `json:"gallery"`
}

type GalleryImageResponse struct {
	ImageData string `json:"image_data"`
	Filename  string `json:"filename"`
	Size      int    `json:"size"`
}

type GalleryRequest struct {
	GalleryName string `json:"gallery_name"`
}

type GalleryImageRequest struct {
	GalleryName string `json:"gallery_name"`
	ImageName   string `json:"image_name"`
}

type GalleriesRequest struct {
	// Currently no additional fields needed
}

// Constants for message types
const (
	MessageTypeGetInfo               = "getInfo"
	MessageTypeGetInfoResp           = "getInfoResp"
	MessageTypeDiscovery             = "discovery"
	MessageTypeDiscoveryResp         = "discoveryResp"
	MessageTypeGetPeerList           = "getPeerList"
	MessageTypeGetPeerListResp       = "getPeerListResp"
	MessageTypeHolePunchAssist       = "holePunchAssist"
	MessageTypeHolePunchResp         = "holePunchResp"
	MessageTypeGetDocs               = "getDocs"
	MessageTypeGetDocsResp           = "getDocsResp"
	MessageTypeGetDoc                = "getDoc"
	MessageTypeGetDocResp            = "getDocResp"
	MessageTypeGetFiles              = "getFiles"
	MessageTypeGetFilesResp          = "getFilesResp"
	MessageTypeGetGalleries          = "getGalleries"
	MessageTypeGetGalleriesResp      = "getGalleriesResp"
	MessageTypeGetGallery            = "getGallery"
	MessageTypeGetGalleryResp        = "getGalleryResp"
	MessageTypeGetGalleryImage       = "getGalleryImage"
	MessageTypeGetGalleryImageResp   = "getGalleryImageResp"
	MessageTypeGetMediaGalleries     = "getMediaGalleries"
	MessageTypeGetMediaGalleriesResp = "getMediaGalleriesResp"
	MessageTypeGetMediaGallery       = "getMediaGallery"
	MessageTypeGetMediaGalleryResp   = "getMediaGalleryResp"
	MessageTypeGetMediaFile          = "getMediaFile"
	MessageTypeGetMediaFileResp      = "getMediaFileResp"
)
