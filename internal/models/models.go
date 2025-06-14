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

// Gallery represents a photo gallery (subdirectory in images/)
type Gallery struct {
	Name       string   `json:"name"`
	ImageCount int      `json:"image_count"`
	Images     []string `json:"images"`
}

// AudioGallery represents an audio collection (subdirectory in audio/)
type AudioGallery struct {
	Name       string   `json:"name"`
	AudioCount int      `json:"audio_count"`
	AudioFiles []string `json:"audio_files"`
}

// VideoGallery represents a video collection (subdirectory in video/)
type VideoGallery struct {
	Name       string   `json:"name"`
	VideoCount int      `json:"video_count"`
	VideoFiles []string `json:"video_files"`
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

// GalleriesRequest represents a P2P request for galleries list
type GalleriesRequest struct {
	// Currently no additional fields needed
}

// GalleriesResponse represents a P2P response with galleries list
type GalleriesResponse struct {
	Galleries []Gallery `json:"galleries"`
	Count     int       `json:"count"`
}

// GalleryRequest represents a P2P request for a specific gallery
type GalleryRequest struct {
	GalleryName string `json:"gallery_name"`
}

// GalleryResponse represents a P2P response with gallery content
type GalleryResponse struct {
	Gallery *Gallery `json:"gallery"`
}

// GalleryImageRequest represents a P2P request for a specific image
type GalleryImageRequest struct {
	GalleryName string `json:"gallery_name"`
	ImageName   string `json:"image_name"`
}

// GalleryImageResponse represents a P2P response with image data
type GalleryImageResponse struct {
	ImageData string `json:"image_data"` // base64 encoded image data
	Filename  string `json:"filename"`
	Size      int    `json:"size"`
}

// AudioGalleriesRequest represents a P2P request for audio galleries list
type AudioGalleriesRequest struct {
	// Currently no additional fields needed
}

// AudioGalleriesResponse represents a P2P response with audio galleries list
type AudioGalleriesResponse struct {
	AudioGalleries []AudioGallery `json:"audio_galleries"`
	Count          int            `json:"count"`
}

// AudioGalleryRequest represents a P2P request for a specific audio gallery
type AudioGalleryRequest struct {
	GalleryName string `json:"gallery_name"`
}

// AudioGalleryResponse represents a P2P response with audio gallery content
type AudioGalleryResponse struct {
	AudioGallery *AudioGallery `json:"audio_gallery"`
}

// AudioFileRequest represents a P2P request for a specific audio file
type AudioFileRequest struct {
	GalleryName string `json:"gallery_name"`
	AudioName   string `json:"audio_name"`
}

// AudioFileResponse represents a P2P response with audio file data
type AudioFileResponse struct {
	AudioData string `json:"audio_data"` // base64 encoded audio data
	Filename  string `json:"filename"`
	Size      int    `json:"size"`
}

// VideoGalleriesRequest represents a P2P request for video galleries list
type VideoGalleriesRequest struct {
	// Currently no additional fields needed
}

// VideoGalleriesResponse represents a P2P response with video galleries list
type VideoGalleriesResponse struct {
	VideoGalleries []VideoGallery `json:"video_galleries"`
	Count          int            `json:"count"`
}

// VideoGalleryRequest represents a P2P request for a specific video gallery
type VideoGalleryRequest struct {
	GalleryName string `json:"gallery_name"`
}

// VideoGalleryResponse represents a P2P response with video gallery content
type VideoGalleryResponse struct {
	VideoGallery *VideoGallery `json:"video_gallery"`
}

// VideoFileRequest represents a P2P request for a specific video file
type VideoFileRequest struct {
	GalleryName string `json:"gallery_name"`
	VideoName   string `json:"video_name"`
}

// VideoFileResponse represents a P2P response with video file data
type VideoFileResponse struct {
	VideoData string `json:"video_data"` // base64 encoded video data
	Filename  string `json:"filename"`
	Size      int    `json:"size"`
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
	MessageTypeGetAudioGalleries     = "getAudioGalleries"
	MessageTypeGetAudioGalleriesResp = "getAudioGalleriesResp"
	MessageTypeGetAudioGallery       = "getAudioGallery"
	MessageTypeGetAudioGalleryResp   = "getAudioGalleryResp"
	MessageTypeGetAudioFile          = "getAudioFile"
	MessageTypeGetAudioFileResp      = "getAudioFileResp"
	MessageTypeGetVideoGalleries     = "getVideoGalleries"
	MessageTypeGetVideoGalleriesResp = "getVideoGalleriesResp"
	MessageTypeGetVideoGallery       = "getVideoGallery"
	MessageTypeGetVideoGalleryResp   = "getVideoGalleryResp"
	MessageTypeGetVideoFile          = "getVideoFile"
	MessageTypeGetVideoFileResp      = "getVideoFileResp"
)
