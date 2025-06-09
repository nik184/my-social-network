package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"my-social-network/internal/models"
	"my-social-network/internal/services"
)

// Handler manages HTTP requests
type Handler struct {
	appService      *services.AppService
	templateService *services.TemplateService
}

// NewHandler creates a new handler
func NewHandler(appService *services.AppService, templateService *services.TemplateService) *Handler {
	return &Handler{
		appService:      appService,
		templateService: templateService,
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
	if h.appService.GetMonitorService() != nil {
		err := h.appService.GetMonitorService().TriggerManualScan()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// Fallback to direct scan
		folderInfo, err := h.appService.GetDirectoryService().ScanDirectory()
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
	if err := h.appService.GetDirectoryService().CreateDirectory(); err != nil {
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

	nodeInfo, err := h.appService.GetP2PService().DiscoverPeer(req.PeerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodeInfo)
}

// HandlePeers handles GET /api/peers requests
func (h *Handler) HandlePeers(w http.ResponseWriter, r *http.Request) {
	validatedPeers := h.appService.GetP2PService().GetConnectedPeers()
	allPeers := h.appService.GetP2PService().GetAllConnectedPeers()

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
		"applicationPeers":    validatedPeerStrings,      // For backward compatibility
		"peers":               validatedPeerStrings,      // For backward compatibility
		"count":               len(validatedPeerStrings), // For backward compatibility
	})
}

// HandleMonitorStatus handles GET /api/monitor requests
func (h *Handler) HandleMonitorStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"monitoring": h.appService.GetMonitorService() != nil,
	}

	if h.appService.GetMonitorService() != nil {
		status["lastScan"] = h.appService.GetMonitorService().GetLastScanTime()
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

	nodeInfo, err := h.appService.GetP2PService().ConnectByIP(req.IP, req.Port, req.PeerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodeInfo)
}

// HandleConnectionInfo handles GET /api/connection-info requests
func (h *Handler) HandleConnectionInfo(w http.ResponseWriter, r *http.Request) {
	connectionInfo := h.appService.GetP2PService().GetConnectionInfo()
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

	images, err := h.appService.GetDirectoryService().GetAvatarImages()
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
	images, err := h.appService.GetDirectoryService().GetAvatarImages()
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
	avatarDir := h.appService.GetDirectoryService().GetAvatarDirectory()
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
		images, err := h.appService.GetDirectoryService().GetPeerAvatarImages(peerID)
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
		images, err := h.appService.GetDirectoryService().GetPeerAvatarImages(peerID)
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
		peerAvatarDir := h.appService.GetDirectoryService().GetPeerAvatarDirectory(peerID)
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

// HandleDocs handles GET /api/docs requests
func (h *Handler) HandleDocs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	docs, err := h.appService.GetDirectoryService().GetDocs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"docs":  docs,
		"count": len(docs),
	})
}

// HandleDoc handles GET /api/docs/{filename} requests
func (h *Handler) HandleDoc(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract filename from URL path
	filename := r.URL.Path[len("/api/docs/"):]
	if filename == "" {
		http.Error(w, "Filename required", http.StatusBadRequest)
		return
	}

	doc, err := h.appService.GetDirectoryService().GetDoc(filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

// HandleFriends handles GET /api/friends requests
func (h *Handler) HandleFriends(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		friends, err := h.appService.GetDatabaseService().GetFriends()
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

		err := h.appService.GetDatabaseService().AddFriend(req.PeerID, req.PeerName)
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
		friends, err := h.appService.GetDatabaseService().GetFriends()
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
		err := h.appService.GetDatabaseService().RemoveFriend(peerID)
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

// HandlePeerDocs handles GET/POST /api/peer-docs/{peerID} and /api/peer-docs/{peerID}/{filename} requests
func (h *Handler) HandlePeerDocs(w http.ResponseWriter, r *http.Request) {
	// Handle POST requests for downloads
	if r.Method == http.MethodPost {
		h.HandleDownloadPeerContent(w, r)
		return
	}
	
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse URL path to extract peerID and optional filename
	pathParts := strings.Split(r.URL.Path[len("/api/peer-docs/"):], "/")
	if len(pathParts) < 1 || pathParts[0] == "" {
		http.Error(w, "Peer ID is required", http.StatusBadRequest)
		return
	}

	peerID := pathParts[0]

	// If no filename provided, return docs list
	if len(pathParts) == 1 {
		// Request docs list from peer via P2P
		docsResponse, err := h.appService.GetP2PService().RequestPeerDocs(peerID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get docs from peer: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(docsResponse)
		return
	}

	// If filename provided, return specific doc
	filename := strings.Join(pathParts[1:], "/") // Join in case filename has slashes
	if filename == "" {
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	// Request specific doc from peer via P2P
	docResponse, err := h.appService.GetP2PService().RequestPeerDoc(peerID, filename)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get doc from peer: %v", err), http.StatusInternalServerError)
		return
	}

	if docResponse.Doc == nil {
		http.Error(w, "Doc not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(docResponse.Doc)
}

// HandleDownloadPeerContent handles POST /api/peer-docs/{peerID}/download requests
func (h *Handler) HandleDownloadPeerContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse URL path to extract peerID
	pathParts := strings.Split(r.URL.Path[len("/api/peer-docs/"):], "/")
	if len(pathParts) < 2 || pathParts[0] == "" || pathParts[1] != "download" {
		http.Error(w, "Invalid URL format. Use /api/peer-docs/{peerID}/download", http.StatusBadRequest)
		return
	}

	peerID := pathParts[0]

	// Request docs list from peer
	docsResponse, err := h.appService.GetP2PService().RequestPeerDocs(peerID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get docs from peer: %v", err), http.StatusInternalServerError)
		return
	}

	if docsResponse.Docs == nil {
		http.Error(w, "No docs found for peer", http.StatusNotFound)
		return
	}

	// Download and save all docs and images
	downloadResult, err := h.downloadAndSavePeerContent(peerID, docsResponse.Docs)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to download content: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(downloadResult)
}

// downloadAndSavePeerContent downloads and saves all content from a peer
func (h *Handler) downloadAndSavePeerContent(peerID string, docs []models.Doc) (map[string]interface{}, error) {
	// Get path manager from service container
	pathManager := h.appService.GetServiceContainer().GetPathManager()
	if pathManager == nil {
		return nil, fmt.Errorf("path manager not available")
	}

	// Create directories
	docsDir := pathManager.GetPeerDocsPath(peerID)
	imagesDir := pathManager.GetPeerImagesPath(peerID)
	
	// Import utils package for EnsureDir function
	if err := h.ensureDirectories(docsDir, imagesDir); err != nil {
		return nil, fmt.Errorf("failed to create directories: %v", err)
	}

	downloadStats := map[string]interface{}{
		"peer_id":           peerID,
		"docs_downloaded":   0,
		"images_downloaded": 0,
		"errors":           []string{},
		"successful_files": []string{},
	}

	// Download each document
	for _, doc := range docs {
		// Get full document content
		docResponse, err := h.appService.GetP2PService().RequestPeerDoc(peerID, doc.Filename)
		if err != nil {
			errorMsg := fmt.Sprintf("Failed to download %s: %v", doc.Filename, err)
			downloadStats["errors"] = append(downloadStats["errors"].([]string), errorMsg)
			continue
		}

		if docResponse.Doc == nil {
			errorMsg := fmt.Sprintf("Doc content not found for %s", doc.Filename)
			downloadStats["errors"] = append(downloadStats["errors"].([]string), errorMsg)
			continue
		}

		// Determine file type and save to appropriate directory
		var saveDir string
		var statKey string
		
		if h.isImageFile(doc.Filename) {
			saveDir = imagesDir
			statKey = "images_downloaded"
		} else {
			saveDir = docsDir
			statKey = "docs_downloaded"
		}

		// Save file
		filePath := filepath.Join(saveDir, doc.Filename)
		err = h.saveContentToFile(filePath, docResponse.Doc.Content)
		if err != nil {
			errorMsg := fmt.Sprintf("Failed to save %s: %v", doc.Filename, err)
			downloadStats["errors"] = append(downloadStats["errors"].([]string), errorMsg)
			continue
		}

		// Update stats
		downloadStats[statKey] = downloadStats[statKey].(int) + 1
		downloadStats["successful_files"] = append(downloadStats["successful_files"].([]string), doc.Filename)
	}

	return downloadStats, nil
}

// ensureDirectories creates the necessary directories
func (h *Handler) ensureDirectories(dirs ...string) error {
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

// isImageFile determines if a file is an image based on its extension
func (h *Handler) isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg", ".ico"}
	for _, imgExt := range imageExts {
		if ext == imgExt {
			return true
		}
	}
	return false
}

// saveContentToFile saves content to a file
func (h *Handler) saveContentToFile(filePath, content string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

// HandleGalleries handles GET /api/galleries requests
func (h *Handler) HandleGalleries(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	galleries, err := h.appService.GetDirectoryService().GetGalleries()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"galleries": galleries,
		"count":     len(galleries),
	})
}

// HandleGalleryImage handles GET /api/galleries/{galleryName} and /api/galleries/{galleryName}/{filename} requests
func (h *Handler) HandleGalleryImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse URL path to extract gallery name and optional filename
	pathParts := strings.Split(r.URL.Path[len("/api/galleries/"):], "/")
	if len(pathParts) < 1 || pathParts[0] == "" {
		http.Error(w, "Gallery name required", http.StatusBadRequest)
		return
	}

	galleryName := pathParts[0]

	// If only gallery name is provided, return images list
	if len(pathParts) == 1 {
		images, err := h.appService.GetDirectoryService().GetGalleryImages(galleryName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"gallery": galleryName,
			"images":  images,
			"count":   len(images),
		})
		return
	}

	// If gallery name and filename are provided, serve the image
	filename := strings.Join(pathParts[1:], "/") // Join in case filename has slashes

	// Validate gallery name and filename to prevent directory traversal
	if strings.Contains(galleryName, "..") || strings.Contains(galleryName, "/") || strings.Contains(galleryName, "\\") {
		http.Error(w, "Invalid gallery name", http.StatusBadRequest)
		return
	}
	if strings.Contains(filename, "..") || strings.Contains(filename, "\\") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	// Get gallery images list to verify the file exists
	images, err := h.appService.GetDirectoryService().GetGalleryImages(galleryName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if the requested file exists in the gallery
	found := false
	for _, img := range images {
		if img == filename {
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Image not found in gallery", http.StatusNotFound)
		return
	}

	// Serve the file
	galleryDir := filepath.Join(h.appService.GetDirectoryService().GetDirectoryPath(), "images", galleryName)
	filePath := filepath.Join(galleryDir, filename)

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

// Page handlers
func (h *Handler) HandleNetworkPage(w http.ResponseWriter, r *http.Request) {
	data := services.TemplateData{
		PageTitle:   "Distributed Social Network",
		CurrentPage: "network",
	}
	h.templateService.RenderPage(w, "network", data)
}

func (h *Handler) HandleProfilePage(w http.ResponseWriter, r *http.Request) {
	data := services.TemplateData{
		PageTitle:   "Profile",
		CurrentPage: "profile",
	}
	h.templateService.RenderPage(w, "profile", data)
}

func (h *Handler) HandleFriendsPage(w http.ResponseWriter, r *http.Request) {
	data := services.TemplateData{
		PageTitle:   "Friends",
		CurrentPage: "friends",
	}
	h.templateService.RenderPage(w, "friends", data)
}

func (h *Handler) HandleFriendProfilePage(w http.ResponseWriter, r *http.Request) {
	data := services.TemplateData{
		PageTitle:   "Friend Profile",
		CurrentPage: "friends", // Keep friends nav active
	}
	h.templateService.RenderPage(w, "profile", data)
}

// RegisterRoutes registers all HTTP routes
func (h *Handler) RegisterRoutes() {
	// Page routes
	http.HandleFunc("/", h.HandleNetworkPage)
	http.HandleFunc("/network", h.HandleNetworkPage)
	http.HandleFunc("/profile", h.HandleProfilePage)
	http.HandleFunc("/friends", h.HandleFriendsPage)
	http.HandleFunc("/friend-profile", h.HandleFriendProfilePage)

	// API routes
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
	http.HandleFunc("/api/docs", h.HandleDocs)
	http.HandleFunc("/api/docs/", h.HandleDoc)
	http.HandleFunc("/api/friends", h.HandleFriends)
	http.HandleFunc("/api/friends/", h.HandleFriend)
	http.HandleFunc("/api/peer-docs/", h.HandlePeerDocs)
	http.HandleFunc("/api/galleries", h.HandleGalleries)
	http.HandleFunc("/api/galleries/", h.HandleGalleryImage)
}
