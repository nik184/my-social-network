package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"my-social-network/internal/models"
	"my-social-network/internal/services"
)

// Handler manages HTTP requests
type Handler struct {
	appService *services.AppService
}

// NewHandler creates a new handler
func NewHandler(appService *services.AppService) *Handler {
	return &Handler{
		appService: appService,
	}
}

// HandleGetInfo handles GET /api/info requests
func (h *Handler) HandleGetInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.appService.GetNodeInfo())
}

// HandleScan handles POST /api/scan requests
func (h *Handler) HandleScan(w http.ResponseWriter, r *http.Request) {
	// Use the monitor service for manual scan if available
	if h.appService.MonitorService != nil {
		err := h.appService.MonitorService.TriggerManualScan()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// Fallback to direct scan
		folderInfo, err := h.appService.DirectoryService.ScanDirectory()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.appService.SetFolderInfo(folderInfo)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.StatusResponse{Status: "success"})
}

// HandleCreate handles POST /api/create requests
func (h *Handler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	if err := h.appService.DirectoryService.CreateDirectory(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.StatusResponse{Status: "directory created"})
}

// HandleDiscover handles POST /api/discover requests
func (h *Handler) HandleDiscover(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req models.DiscoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	nodeInfo, err := h.appService.P2PService.DiscoverPeer(req.PeerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodeInfo)
}

// HandlePeers handles GET /api/peers requests
func (h *Handler) HandlePeers(w http.ResponseWriter, r *http.Request) {
	validatedPeers := h.appService.P2PService.GetConnectedPeers()
	allPeers := h.appService.P2PService.GetAllConnectedPeers()
	
	// Convert peer IDs to strings for JSON
	validatedPeerStrings := make([]string, len(validatedPeers))
	for i, peer := range validatedPeers {
		validatedPeerStrings[i] = peer.String()
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"validatedPeers":      validatedPeerStrings,
		"validatedCount":      len(validatedPeerStrings),
		"totalConnectedCount": len(allPeers),
		"applicationPeers":    validatedPeerStrings, // For backward compatibility
		"peers":               validatedPeerStrings, // For backward compatibility
		"count":               len(validatedPeerStrings), // For backward compatibility
	})
}

// HandleMonitorStatus handles GET /api/monitor requests
func (h *Handler) HandleMonitorStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"monitoring": h.appService.MonitorService != nil,
	}
	
	if h.appService.MonitorService != nil {
		status["lastScan"] = h.appService.MonitorService.GetLastScanTime()
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// HandleConnectByIP handles POST /api/connect-ip requests
func (h *Handler) HandleConnectByIP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req models.IPConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	nodeInfo, err := h.appService.P2PService.ConnectByIP(req.IP, req.Port, req.PeerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodeInfo)
}

// HandleConnectionInfo handles GET /api/connection-info requests
func (h *Handler) HandleConnectionInfo(w http.ResponseWriter, r *http.Request) {
	connectionInfo := h.appService.P2PService.GetConnectionInfo()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(connectionInfo)
}

// HandleConnectionHistory handles GET /api/connection-history requests
func (h *Handler) HandleConnectionHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	history, err := h.appService.GetConnectionHistory()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// HandleSecondDegreePeers handles GET /api/second-degree-peers requests
func (h *Handler) HandleSecondDegreePeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	secondDegreePeers, err := h.appService.GetSecondDegreeConnections()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(secondDegreePeers)
}

// HandleConnectSecondDegree handles POST /api/connect-second-degree requests
func (h *Handler) HandleConnectSecondDegree(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req models.SecondDegreeConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	nodeInfo, err := h.appService.ConnectToSecondDegreePeer(req.TargetPeerID, req.ViaPeerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodeInfo)
}

// HandleAvatarList handles GET /api/avatar requests
func (h *Handler) HandleAvatarList(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	images, err := h.appService.DirectoryService.GetAvatarImages()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"images": images,
		"count":  len(images),
	}

	if len(images) > 0 {
		response["primary"] = images[0] // First image is the primary avatar
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleAvatarImage handles GET /api/avatar/{filename} requests
func (h *Handler) HandleAvatarImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract filename from URL path
	filename := r.URL.Path[len("/api/avatar/"):]
	if filename == "" {
		http.Error(w, "Filename required", http.StatusBadRequest)
		return
	}

	// Get avatar images list to verify the file exists
	images, err := h.appService.DirectoryService.GetAvatarImages()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if the requested file exists in our avatar list
	found := false
	for _, img := range images {
		if img == filename {
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Avatar image not found", http.StatusNotFound)
		return
	}

	// Serve the file
	avatarDir := h.appService.DirectoryService.GetAvatarDirectory()
	filePath := filepath.Join(avatarDir, filename)
	
	// Set appropriate content type based on file extension
	ext := filepath.Ext(filename)
	switch ext {
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".webp":
		w.Header().Set("Content-Type", "image/webp")
	case ".bmp":
		w.Header().Set("Content-Type", "image/bmp")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	http.ServeFile(w, r, filePath)
}

// HandlePeerAvatar handles GET /api/peer-avatar/{peerID} and /api/peer-avatar/{peerID}/{filename} requests
func (h *Handler) HandlePeerAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract path segments from URL
	pathSegments := r.URL.Path[len("/api/peer-avatar/"):]
	if pathSegments == "" {
		http.Error(w, "Peer ID required", http.StatusBadRequest)
		return
	}

	// Split path into components
	pathParts := make([]string, 0)
	currentPart := ""
	for _, char := range pathSegments {
		if char == '/' {
			if currentPart != "" {
				pathParts = append(pathParts, currentPart)
				currentPart = ""
			}
		} else {
			currentPart += string(char)
		}
	}
	if currentPart != "" {
		pathParts = append(pathParts, currentPart)
	}

	if len(pathParts) == 0 {
		http.Error(w, "Peer ID required", http.StatusBadRequest)
		return
	}

	peerID := pathParts[0]

	// If only peer ID is provided, return avatar list
	if len(pathParts) == 1 {
		images, err := h.appService.DirectoryService.GetPeerAvatarImages(peerID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"peer_id": peerID,
			"images":  images,
			"count":   len(images),
		}

		if len(images) > 0 {
			response["primary"] = images[0] // First image is the primary avatar
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// If peer ID and filename are provided, serve the image
	if len(pathParts) >= 2 {
		filename := pathParts[1]

		// Get peer avatar images list to verify the file exists
		images, err := h.appService.DirectoryService.GetPeerAvatarImages(peerID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Check if the requested file exists in the peer's avatar list
		found := false
		for _, img := range images {
			if img == filename {
				found = true
				break
			}
		}

		if !found {
			http.Error(w, "Peer avatar image not found", http.StatusNotFound)
			return
		}

		// Serve the file
		peerAvatarDir := h.appService.DirectoryService.GetPeerAvatarDirectory(peerID)
		filePath := filepath.Join(peerAvatarDir, filename)
		
		// Set appropriate content type based on file extension
		ext := filepath.Ext(filename)
		switch ext {
		case ".jpg", ".jpeg":
			w.Header().Set("Content-Type", "image/jpeg")
		case ".png":
			w.Header().Set("Content-Type", "image/png")
		case ".gif":
			w.Header().Set("Content-Type", "image/gif")
		case ".webp":
			w.Header().Set("Content-Type", "image/webp")
		case ".bmp":
			w.Header().Set("Content-Type", "image/bmp")
		default:
			w.Header().Set("Content-Type", "application/octet-stream")
		}

		http.ServeFile(w, r, filePath)
		return
	}

	http.Error(w, "Invalid request path", http.StatusBadRequest)
}

// HandleNotes handles GET /api/notes requests
func (h *Handler) HandleNotes(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	notes, err := h.appService.DirectoryService.GetNotes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"notes": notes,
		"count": len(notes),
	})
}

// HandleNote handles GET /api/notes/{filename} requests
func (h *Handler) HandleNote(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract filename from URL path
	filename := r.URL.Path[len("/api/notes/"):]
	if filename == "" {
		http.Error(w, "Filename required", http.StatusBadRequest)
		return
	}

	note, err := h.appService.DirectoryService.GetNote(filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(note)
}

// HandleFriends handles GET /api/friends requests
func (h *Handler) HandleFriends(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		friends, err := h.appService.DatabaseService.GetFriends()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := models.FriendsResponse{
			Friends: friends,
			Count:   len(friends),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	case http.MethodPost:
		var req models.AddFriendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.PeerID == "" || req.PeerName == "" {
			http.Error(w, "peer_id and peer_name are required", http.StatusBadRequest)
			return
		}

		err := h.appService.DatabaseService.AddFriend(req.PeerID, req.PeerName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.StatusResponse{Status: "success", Message: "Friend added successfully"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleFriend handles GET and DELETE /api/friends/{peerID} requests
func (h *Handler) HandleFriend(w http.ResponseWriter, r *http.Request) {
	// Extract peer ID from URL path
	peerID := r.URL.Path[len("/api/friends/"):]
	if peerID == "" {
		http.Error(w, "Peer ID is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get specific friend info
		friends, err := h.appService.DatabaseService.GetFriends()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Find the friend with matching peer ID
		for _, friend := range friends {
			if friend.PeerID == peerID {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(friend)
				return
			}
		}

		http.Error(w, "Friend not found", http.StatusNotFound)

	case http.MethodDelete:
		// Remove friend
		err := h.appService.DatabaseService.RemoveFriend(peerID)
		if err != nil {
			if err.Error() == "friend not found" {
				http.Error(w, err.Error(), http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.StatusResponse{Status: "success", Message: "Friend removed successfully"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandlePeerNotes handles GET /api/peer-notes/{peerID} and /api/peer-notes/{peerID}/{filename} requests
func (h *Handler) HandlePeerNotes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse URL path to extract peerID and optional filename
	pathParts := strings.Split(r.URL.Path[len("/api/peer-notes/"):], "/")
	if len(pathParts) < 1 || pathParts[0] == "" {
		http.Error(w, "Peer ID is required", http.StatusBadRequest)
		return
	}

	peerID := pathParts[0]

	// If no filename provided, return notes list
	if len(pathParts) == 1 {
		// Request notes list from peer via P2P
		notesResponse, err := h.appService.P2PService.RequestPeerNotes(peerID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get notes from peer: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(notesResponse)
		return
	}

	// If filename provided, return specific note
	filename := strings.Join(pathParts[1:], "/") // Join in case filename has slashes
	if filename == "" {
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	// Request specific note from peer via P2P
	noteResponse, err := h.appService.P2PService.RequestPeerNote(peerID, filename)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get note from peer: %v", err), http.StatusInternalServerError)
		return
	}

	if noteResponse.Note == nil {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(noteResponse.Note)
}

// RegisterRoutes registers all HTTP routes
func (h *Handler) RegisterRoutes() {
	http.HandleFunc("/api/info", h.HandleGetInfo)
	http.HandleFunc("/api/scan", h.HandleScan)
	http.HandleFunc("/api/create", h.HandleCreate)
	http.HandleFunc("/api/discover", h.HandleDiscover)
	http.HandleFunc("/api/peers", h.HandlePeers)
	http.HandleFunc("/api/monitor", h.HandleMonitorStatus)
	http.HandleFunc("/api/connect-ip", h.HandleConnectByIP)
	http.HandleFunc("/api/connection-info", h.HandleConnectionInfo)
	http.HandleFunc("/api/connection-history", h.HandleConnectionHistory)
	http.HandleFunc("/api/second-degree-peers", h.HandleSecondDegreePeers)
	http.HandleFunc("/api/connect-second-degree", h.HandleConnectSecondDegree)
	http.HandleFunc("/api/avatar", h.HandleAvatarList)
	http.HandleFunc("/api/avatar/", h.HandleAvatarImage)
	http.HandleFunc("/api/peer-avatar/", h.HandlePeerAvatar)
	http.HandleFunc("/api/notes", h.HandleNotes)
	http.HandleFunc("/api/notes/", h.HandleNote)
	http.HandleFunc("/api/friends", h.HandleFriends)
	http.HandleFunc("/api/friends/", h.HandleFriend)
	http.HandleFunc("/api/peer-notes/", h.HandlePeerNotes)
}