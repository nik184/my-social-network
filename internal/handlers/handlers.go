package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"old-school/internal/models"
	"old-school/internal/services"
	"old-school/internal/utils"
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

// HandlePeerFriends handles GET /api/peer-friends/{peerID} requests
func (h *Handler) HandlePeerFriends(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract peer ID from URL path
	peerID := r.URL.Path[len("/api/peer-friends/"):]
	if peerID == "" {
		http.Error(w, "Peer ID is required", http.StatusBadRequest)
		return
	}

	// First, try to get friends from local database
	friends, err := h.appService.GetDatabaseService().GetPeerFriends(peerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If no friends found locally, try to fetch from remote peer
	if len(friends) == 0 {
		log.Printf("🔍 No local friends data for peer %s, attempting remote fetch...", peerID)
		
		p2pService := h.appService.GetP2PService()
		if p2pService != nil {
			remoteFriends, err := p2pService.FetchAndSavePeerFriends(peerID)
			if err != nil {
				log.Printf("⚠️ Failed to fetch friends from remote peer %s: %v", peerID, err)
				// Continue with empty friends list - don't fail the request
			} else if len(remoteFriends) > 0 {
				log.Printf("✅ Successfully fetched and saved %d friends from remote peer %s", len(remoteFriends), peerID)
				friends = remoteFriends
			}
		}
	}

	response := models.FriendsResponse{
		Friends: friends,
		Count:   len(friends),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
		"errors":            []string{},
		"successful_files":  []string{},
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

// Page handlers

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

// isValidDocumentFile checks if a file is a valid document type
func (h *Handler) isValidDocumentFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := []string{".md", ".pdf", ".txt", ".html", ".djvu", ".doc", ".docx"}
	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

// isValidImageFile checks if a file is a valid image type
func (h *Handler) isValidImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg"}
	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

// HandleSyncFriendFiles handles POST /api/sync-friend-files requests
func (h *Handler) HandleSyncFriendFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	friendService := h.appService.GetFriendService()
	if friendService == nil {
		http.Error(w, "Friend service not available", http.StatusInternalServerError)
		return
	}

	// Check if specific peer ID is provided
	peerID := r.URL.Query().Get("peer_id")

	var err error
	if peerID != "" {
		// Sync specific friend
		err = friendService.SyncSpecificFriendFiles(peerID)
	} else {
		// Sync all friends
		err = friendService.SyncFriendFilesMetadata()
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Files metadata sync completed successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandlePeerGalleries handles GET /api/peer-galleries/{peerID} and other peer gallery requests
func (h *Handler) HandlePeerGalleries(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse URL path to extract peerID and optional gallery/image path
	pathParts := strings.Split(r.URL.Path[len("/api/peer-galleries/"):], "/")
	if len(pathParts) < 1 || pathParts[0] == "" {
		http.Error(w, "Peer ID is required", http.StatusBadRequest)
		return
	}

	peerID := pathParts[0]

	// Route based on path length
	switch len(pathParts) {
	case 1:
		// GET /api/peer-galleries/{peerID} - list galleries
		h.handlePeerGalleriesList(w, r, peerID)
	case 2:
		// GET /api/peer-galleries/{peerID}/{galleryName} - get gallery details
		galleryName := pathParts[1]
		h.handlePeerGalleryDetails(w, r, peerID, galleryName)
	case 3:
		// GET /api/peer-galleries/{peerID}/{galleryName}/{imageName} - get/download image
		galleryName := pathParts[1]
		imageName := pathParts[2]
		h.handlePeerGalleryImage(w, r, peerID, galleryName, imageName)
	default:
		http.Error(w, "Invalid request path", http.StatusBadRequest)
	}
}

// handlePeerGalleriesList handles requests for a peer's galleries list
func (h *Handler) handlePeerGalleriesList(w http.ResponseWriter, r *http.Request, peerID string) {
	// Request galleries list from peer via P2P
	galleriesResponse, err := h.appService.GetP2PService().RequestPeerGalleries(peerID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get galleries from peer: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(galleriesResponse)
}

// handlePeerGalleryDetails handles requests for a specific peer gallery
func (h *Handler) handlePeerGalleryDetails(w http.ResponseWriter, r *http.Request, peerID, galleryName string) {
	// Request specific gallery from peer via P2P
	galleryResponse, err := h.appService.GetP2PService().RequestPeerGallery(peerID, galleryName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get gallery from peer: %v", err), http.StatusInternalServerError)
		return
	}

	if galleryResponse.Gallery == nil {
		http.Error(w, "Gallery not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(galleryResponse.Gallery)
}

// handlePeerGalleryImage handles requests for a specific image from a peer's gallery
func (h *Handler) handlePeerGalleryImage(w http.ResponseWriter, r *http.Request, peerID, galleryName, imageName string) {
	// Check if image is already cached locally
	cachedPath := h.getCachedImagePath(peerID, galleryName, imageName)
	if cachedPath != "" {
		// Serve cached image
		h.serveCachedImage(w, r, cachedPath, imageName)
		return
	}

	// Request image from peer via P2P and cache it
	imageResponse, err := h.appService.GetP2PService().RequestPeerGalleryImage(peerID, galleryName, imageName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get image from peer: %v", err), http.StatusInternalServerError)
		return
	}

	if imageResponse.ImageData == "" {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}

	// Decode base64 image data
	imageData, err := base64.StdEncoding.DecodeString(imageResponse.ImageData)
	if err != nil {
		http.Error(w, "Failed to decode image data", http.StatusInternalServerError)
		return
	}

	// Download and save the image locally
	if err := h.downloadImage(peerID, galleryName, imageName, imageData); err != nil {
		log.Printf("Warning: Failed to save downloaded image: %v", err)
	}

	// Set appropriate content type based on file extension
	ext := filepath.Ext(imageName)
	switch strings.ToLower(ext) {
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
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	// Serve the image
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(imageData)))
	w.Write(imageData)
}

// getCachedImagePath checks if an image is already downloaded locally
func (h *Handler) getCachedImagePath(peerID, galleryName, imageName string) string {
	// Get path manager from service container
	pathManager := h.appService.GetServiceContainer().GetPathManager()
	if pathManager == nil {
		return ""
	}

	// Create path for downloaded peer images in gallery structure
	galleryDir := pathManager.GetPeerGalleryPath(peerID, galleryName)
	imagePath := filepath.Join(galleryDir, imageName)

	// Check if file exists
	if _, err := os.Stat(imagePath); err == nil {
		return imagePath
	}

	return ""
}

// serveCachedImage serves a cached image from local storage
func (h *Handler) serveCachedImage(w http.ResponseWriter, r *http.Request, imagePath, imageName string) {
	// Set appropriate content type based on file extension
	ext := filepath.Ext(imageName)
	switch strings.ToLower(ext) {
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
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	http.ServeFile(w, r, imagePath)
}

// downloadImage saves an image to the downloaded folder structure
func (h *Handler) downloadImage(peerID, galleryName, imageName string, imageData []byte) error {
	// Get path manager from service container
	pathManager := h.appService.GetServiceContainer().GetPathManager()
	if pathManager == nil {
		return fmt.Errorf("path manager not available")
	}

	// Create gallery directory in downloaded structure
	galleryDir := pathManager.GetPeerGalleryPath(peerID, galleryName)
	if err := os.MkdirAll(galleryDir, 0755); err != nil {
		return fmt.Errorf("failed to create gallery directory: %v", err)
	}

	// Save image to downloaded folder
	imagePath := filepath.Join(galleryDir, imageName)
	if err := os.WriteFile(imagePath, imageData, 0644); err != nil {
		return fmt.Errorf("failed to write downloaded image: %v", err)
	}

	log.Printf("📷 Downloaded image %s for peer %s in gallery %s", imageName, peerID, galleryName)
	return nil
}

// HandleDownloadedContent handles serving downloaded peer content
func (h *Handler) HandleDownloadedContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse URL path: /api/downloaded/{peerID}/{type}/{gallery?}/{filename?}
	pathParts := strings.Split(r.URL.Path[len("/api/downloaded/"):], "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid path format. Expected: /api/downloaded/{peerID}/{type}/{gallery?}/{filename?}", http.StatusBadRequest)
		return
	}

	peerID := pathParts[0]
	contentType := pathParts[1] // "images" or "docs"

	switch contentType {
	case "images":
		h.handleDownloadedImages(w, r, peerID, pathParts[2:])
	case "docs":
		h.handleDownloadedDocs(w, r, peerID, pathParts[2:])
	default:
		http.Error(w, "Invalid content type. Use 'images' or 'docs'", http.StatusBadRequest)
	}
}

// handleDownloadedImages handles downloaded image requests
func (h *Handler) handleDownloadedImages(w http.ResponseWriter, r *http.Request, peerID string, pathParts []string) {
	switch len(pathParts) {
	case 0:
		// GET /api/downloaded/{peerID}/images - list galleries
		galleries, err := h.appService.GetDirectoryService().GetPeerMediaGalleries(peerID, models.MediaTypeImage)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get galleries: %v", err), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"galleries": galleries,
			"count":     len(galleries),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	case 1:
		// GET /api/downloaded/{peerID}/images/{galleryName} - list gallery images
		galleryName := pathParts[0]
		images, err := h.appService.GetDirectoryService().GetPeerMediaGalleryFiles(peerID, galleryName, models.MediaTypeImage)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get gallery images: %v", err), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"gallery": galleryName,
			"images":  images,
			"count":   len(images),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	case 2:
		// GET /api/downloaded/{peerID}/images/{galleryName}/{imageName} - serve image
		galleryName := pathParts[0]
		imageName := pathParts[1]

		// Get path manager from service container
		pathManager := h.appService.GetServiceContainer().GetPathManager()
		if pathManager == nil {
			http.Error(w, "Path manager not available", http.StatusInternalServerError)
			return
		}

		// Construct file path
		imagePath := filepath.Join(pathManager.GetPeerGalleryPath(peerID, galleryName), imageName)

		// Validate that file exists and is within the expected directory
		images, err := h.appService.GetDirectoryService().GetPeerMediaGalleryFiles(peerID, galleryName, models.MediaTypeImage)
		if err != nil {
			http.Error(w, "Gallery not found", http.StatusNotFound)
			return
		}

		// Check if requested image exists in the gallery
		found := false
		for _, img := range images {
			if img == imageName {
				found = true
				break
			}
		}

		if !found {
			http.Error(w, "Image not found in gallery", http.StatusNotFound)
			return
		}

		// Set appropriate content type based on file extension
		ext := filepath.Ext(imageName)
		switch strings.ToLower(ext) {
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
		case ".svg":
			w.Header().Set("Content-Type", "image/svg+xml")
		default:
			w.Header().Set("Content-Type", "application/octet-stream")
		}

		// Serve the file
		http.ServeFile(w, r, imagePath)

	default:
		http.Error(w, "Invalid path format", http.StatusBadRequest)
	}
}

// handleDownloadedDocs handles downloaded document requests
func (h *Handler) handleDownloadedDocs(w http.ResponseWriter, r *http.Request, peerID string, pathParts []string) {
	// For now, just return a placeholder - docs download logic can be implemented later
	http.Error(w, "Downloaded docs serving not implemented yet", http.StatusNotImplemented)
}

// HandleDeleteDoc handles DELETE /api/delete/docs/{subdirectory}/{filename} requests
func (h *Handler) HandleDeleteDoc(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract path parts from URL
	pathParts := strings.Split(r.URL.Path[len("/api/delete/docs/"):], "/")
	if len(pathParts) < 2 || pathParts[0] == "" || pathParts[1] == "" {
		http.Error(w, "Subdirectory and filename required", http.StatusBadRequest)
		return
	}

	subdirectory := pathParts[0]
	filename := strings.Join(pathParts[1:], "/") // Join in case filename has slashes

	// Validate subdirectory and filename to prevent directory traversal
	if strings.Contains(subdirectory, "..") || strings.Contains(subdirectory, "/") || strings.Contains(subdirectory, "\\") {
		http.Error(w, "Invalid subdirectory name", http.StatusBadRequest)
		return
	}
	if strings.Contains(filename, "..") || strings.Contains(filename, "\\") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	// Get docs directory and construct file path
	docsDir := filepath.Join(h.appService.GetDirectoryService().GetDirectoryPath(), "docs")
	filePath := filepath.Join(docsDir, subdirectory, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Delete file from filesystem
	if err := os.Remove(filePath); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete file: %v", err), http.StatusInternalServerError)
		return
	}

	// Delete record from database
	relativePath := filepath.Join("docs", subdirectory, filename)
	if err := h.appService.GetDatabaseService().DeleteFileRecordByPath(relativePath); err != nil {
		log.Printf("Warning: Failed to delete file record from database: %v", err)
	}

	log.Printf("🗑️ Deleted document: %s from %s", filename, subdirectory)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"message":      "Document deleted successfully",
		"filename":     filename,
		"subdirectory": subdirectory,
	})
}

// HandleDeleteImage handles DELETE /api/delete/images/{gallery}/{filename} requests
func (h *Handler) HandleDeleteImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse URL path to extract gallery and filename
	pathParts := strings.Split(r.URL.Path[len("/api/delete/images/"):], "/")
	if len(pathParts) < 2 || pathParts[0] == "" || pathParts[1] == "" {
		http.Error(w, "Gallery name and filename required", http.StatusBadRequest)
		return
	}

	galleryName := pathParts[0]
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

	// Get images directory and construct file path
	imagesDir := filepath.Join(h.appService.GetDirectoryService().GetDirectoryPath(), "images")
	filePath := filepath.Join(imagesDir, galleryName, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}

	// Delete file from filesystem
	if err := os.Remove(filePath); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete image: %v", err), http.StatusInternalServerError)
		return
	}

	// Delete record from database
	relativePath := filepath.Join("images", galleryName, filename)
	if err := h.appService.GetDatabaseService().DeleteFileRecordByPath(relativePath); err != nil {
		log.Printf("Warning: Failed to delete image record from database: %v", err)
	}

	log.Printf("🗑️ Deleted image: %s from gallery %s", filename, galleryName)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"message":  "Image deleted successfully",
		"filename": filename,
		"gallery":  galleryName,
	})
}

// HandleDocsSubdirectories handles GET /api/subdirectories/docs requests
func (h *Handler) HandleDocsSubdirectories(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	subdirs, err := h.appService.GetDirectoryService().GetDocsSubdirectories()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get docs subdirectories: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"subdirectories": subdirs,
		"count":          len(subdirs),
	})
}

// HandleMediaGalleries handles GET /api/media/{mediaType}/galleries requests
func (h *Handler) HandleMediaGalleries(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract media type from URL path
	pathParts := strings.Split(r.URL.Path[len("/api/media/"):], "/")
	if len(pathParts) < 2 || pathParts[0] == "" || pathParts[1] != "galleries" {
		http.Error(w, "Invalid path format. Use /api/media/{type}/galleries", http.StatusBadRequest)
		return
	}

	mediaTypeStr := pathParts[0]
	var mediaType models.MediaType

	switch mediaTypeStr {
	case "images", "image":
		mediaType = models.MediaTypeImage
	case "audio":
		mediaType = models.MediaTypeAudio
	case "video":
		mediaType = models.MediaTypeVideo
	case "docs":
		mediaType = models.MediaTypeDocs
	default:
		http.Error(w, "Invalid media type. Use 'image', 'audio', 'video', or 'docs'", http.StatusBadRequest)
		return
	}

	galleries, err := h.appService.GetDirectoryService().GetMediaGalleries(mediaType)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get %s galleries: %v", mediaType, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"media_type": mediaType,
		"galleries":  galleries,
		"count":      len(galleries),
	})
}

// Unified media handlers

// HandleMediaGalleryContent handles GET /api/media/{mediaType}/galleries/{galleryName} and /api/media/{mediaType}/galleries/{galleryName}/{fileName} requests
func (h *Handler) HandleMediaGalleryContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse URL path
	pathParts := strings.Split(r.URL.Path[len("/api/media/"):], "/")
	if len(pathParts) < 3 || pathParts[0] == "" || pathParts[1] != "galleries" || pathParts[2] == "" {
		http.Error(w, "Invalid path format. Use /api/media/{type}/galleries/{gallery}/{file?}", http.StatusBadRequest)
		return
	}

	mediaTypeStr := pathParts[0]
	galleryName := pathParts[2]

	var mediaType models.MediaType
	switch mediaTypeStr {
	case "images", "image":
		mediaType = models.MediaTypeImage
	case "audio":
		mediaType = models.MediaTypeAudio
	case "video":
		mediaType = models.MediaTypeVideo
	case "docs":
		mediaType = models.MediaTypeDocs
	default:
		http.Error(w, "Invalid media type. Use 'image', 'audio', 'video', or 'docs'", http.StatusBadRequest)
		return
	}

	// If only gallery name is provided, return files list
	if len(pathParts) == 3 {
		files, err := h.appService.GetDirectoryService().GetMediaGalleryFiles(mediaType, galleryName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get %s gallery files: %v", mediaType, err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"media_type": mediaType,
			"gallery":    galleryName,
			"files":      files,
			"count":      len(files),
		})
		return
	}

	// If filename is provided, serve the file
	fileName := strings.Join(pathParts[3:], "/")

	// Validate file type
	var isValidFileFunc func(string) bool
	switch mediaType {
	case models.MediaTypeImage:
		isValidFileFunc = h.isValidImageFile
	case models.MediaTypeAudio:
		isValidFileFunc = utils.IsAudioFile
	case models.MediaTypeVideo:
		isValidFileFunc = utils.IsVideoFile
	case models.MediaTypeDocs:
		isValidFileFunc = h.isValidDocumentFile
	default:
		http.Error(w, "Unsupported media type", http.StatusBadRequest)
		return
	}

	if !isValidFileFunc(fileName) {
		http.Error(w, fmt.Sprintf("Invalid %s file", mediaType), http.StatusBadRequest)
		return
	}

	// Get files list to verify the file exists
	files, err := h.appService.GetDirectoryService().GetMediaGalleryFiles(mediaType, galleryName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get %s gallery files: %v", mediaType, err), http.StatusInternalServerError)
		return
	}

	// Check if the requested file exists
	found := false
	for _, file := range files {
		if file == fileName {
			found = true
			break
		}
	}

	if !found {
		http.Error(w, fmt.Sprintf("%s file not found in gallery", strings.Title(string(mediaType))), http.StatusNotFound)
		return
	}

	// Build file path
	var filePath string
	baseDir := h.appService.GetDirectoryService().GetDirectoryPath()

	switch mediaType {
	case models.MediaTypeImage:
		if galleryName == "root_images" {
			filePath = h.findFileInMediaDirectory(baseDir, "images", fileName)
		} else {
			filePath = filepath.Join(baseDir, "images", galleryName, fileName)
		}
	case models.MediaTypeAudio:
		if galleryName == "root_audio" {
			filePath = h.findFileInMediaDirectory(baseDir, "audio", fileName)
		} else {
			filePath = filepath.Join(baseDir, "audio", galleryName, fileName)
		}
	case models.MediaTypeVideo:
		if galleryName == "root_video" {
			filePath = h.findFileInMediaDirectory(baseDir, "video", fileName)
		} else {
			filePath = filepath.Join(baseDir, "video", galleryName, fileName)
		}
	case models.MediaTypeDocs:
		if galleryName == "root_docs" {
			filePath = h.findFileInMediaDirectory(baseDir, "docs", fileName)
		} else {
			filePath = filepath.Join(baseDir, "docs", galleryName, fileName)
		}
	}

	// Set appropriate content type
	h.setMediaContentType(w, fileName, mediaType)

	// Serve the file
	// Check if file exists
	if filePath == "" {
		http.Error(w, fmt.Sprintf("%s file not found", strings.Title(string(mediaType))), http.StatusNotFound)
		return
	}

	// For HTML files in docs, sanitize before serving
	if mediaType == models.MediaTypeDocs && (strings.ToLower(filepath.Ext(fileName)) == ".html" || strings.ToLower(filepath.Ext(fileName)) == ".htm") {
		// Read the file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			http.Error(w, "Failed to read file", http.StatusInternalServerError)
			return
		}
		
		// Sanitize HTML content
		sanitizedContent := h.sanitizeHTML(string(content))
		
		// Serve sanitized content
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(sanitizedContent)))
		w.Write([]byte(sanitizedContent))
		return
	}

	// For non-HTML files, serve directly
	http.ServeFile(w, r, filePath)
}

// findFileInMediaDirectory searches for a file in the main directory and all subdirectories
func (h *Handler) findFileInMediaDirectory(baseDir, mediaType, fileName string) string {
	mediaDir := filepath.Join(baseDir, mediaType)

	// First check in the main directory
	mainFilePath := filepath.Join(mediaDir, fileName)
	if _, err := os.Stat(mainFilePath); err == nil {
		return mainFilePath
	}

	// Then check in all subdirectories
	files, err := os.ReadDir(mediaDir)
	if err != nil {
		return ""
	}

	for _, file := range files {
		if file.IsDir() {
			subDirPath := filepath.Join(mediaDir, file.Name(), fileName)
			if _, err := os.Stat(subDirPath); err == nil {
				return subDirPath
			}
		}
	}

	return ""
}

// HandleUploadMedia handles POST /api/media/{mediaType}/upload requests
func (h *Handler) HandleUploadMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract media type from URL path
	pathParts := strings.Split(r.URL.Path[len("/api/media/"):], "/")
	if len(pathParts) < 2 || pathParts[0] == "" || pathParts[1] != "upload" {
		http.Error(w, "Invalid path format. Use /api/media/{type}/upload", http.StatusBadRequest)
		return
	}

	mediaTypeStr := pathParts[0]
	var mediaType models.MediaType
	var maxMemory int64
	var isValidFileFunc func(string) bool

	switch mediaTypeStr {
	case "images", "image":
		mediaType = models.MediaTypeImage
		maxMemory = 32 << 20 // 32MB
		isValidFileFunc = h.isValidImageFile
	case "audio":
		mediaType = models.MediaTypeAudio
		maxMemory = 100 << 20 // 100MB
		isValidFileFunc = utils.IsAudioFile
	case "video":
		mediaType = models.MediaTypeVideo
		maxMemory = 500 << 20 // 500MB
		isValidFileFunc = utils.IsVideoFile
	case "docs":
		mediaType = models.MediaTypeDocs
		maxMemory = 32 << 20 // 32MB
		isValidFileFunc = h.isValidDocumentFile
	default:
		http.Error(w, "Invalid media type. Use 'image', 'audio', 'video', 'docs', or 'avatar'", http.StatusBadRequest)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(maxMemory)
	if err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get uploaded files
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		http.Error(w, "No files uploaded", http.StatusBadRequest)
		return
	}

	// Get subdirectory (optional)
	subdirectory := strings.TrimSpace(r.FormValue("subdirectory"))
	if subdirectory != "" {
		subdirectory = filepath.Clean(subdirectory)
		if strings.Contains(subdirectory, "..") || strings.HasPrefix(subdirectory, "/") {
			http.Error(w, "Invalid subdirectory path", http.StatusBadRequest)
			return
		}
	}

	// Create target directory
	baseDir := h.appService.GetDirectoryService().GetDirectoryPath()
	var mediaDir string

	switch mediaType {
	case models.MediaTypeImage:
		mediaDir = filepath.Join(baseDir, "images")
	case models.MediaTypeAudio:
		mediaDir = filepath.Join(baseDir, "audio")
	case models.MediaTypeVideo:
		mediaDir = filepath.Join(baseDir, "video")
	case models.MediaTypeDocs:
		mediaDir = filepath.Join(baseDir, "docs")
	}

	targetDir := mediaDir
	if subdirectory != "" {
		targetDir = filepath.Join(mediaDir, subdirectory)
	}

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		http.Error(w, "Failed to create directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Process uploaded files
	var uploadedFiles []string
	var errors []string

	for _, fileHeader := range files {
		// Validate file type
		if !isValidFileFunc(fileHeader.Filename) {
			errors = append(errors, fmt.Sprintf("Invalid file type: %s", fileHeader.Filename))
			continue
		}

		// Open uploaded file
		file, err := fileHeader.Open()
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to open %s: %v", fileHeader.Filename, err))
			continue
		}
		defer file.Close()

		// Create destination file
		destPath := filepath.Join(targetDir, fileHeader.Filename)
		destFile, err := os.Create(destPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to create %s: %v", fileHeader.Filename, err))
			continue
		}
		defer destFile.Close()

		// Copy file content
		_, err = io.Copy(destFile, file)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to save %s: %v", fileHeader.Filename, err))
			os.Remove(destPath)
			continue
		}

		uploadedFiles = append(uploadedFiles, fileHeader.Filename)
	}

	// Return response
	response := map[string]interface{}{
		"success":        len(uploadedFiles) > 0,
		"media_type":     mediaType,
		"uploaded_files": uploadedFiles,
		"uploaded_count": len(uploadedFiles),
		"errors":         errors,
		"target_dir":     targetDir,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// setMediaContentType sets the appropriate content type for media files
func (h *Handler) setMediaContentType(w http.ResponseWriter, fileName string, mediaType models.MediaType) {
	ext := strings.ToLower(filepath.Ext(fileName))

	switch mediaType {
	case models.MediaTypeImage:
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
		case ".svg":
			w.Header().Set("Content-Type", "image/svg+xml")
		default:
			w.Header().Set("Content-Type", "application/octet-stream")
		}
	case models.MediaTypeAudio:
		switch ext {
		case ".mp3":
			w.Header().Set("Content-Type", "audio/mpeg")
		case ".wav":
			w.Header().Set("Content-Type", "audio/wav")
		case ".ogg":
			w.Header().Set("Content-Type", "audio/ogg")
		case ".flac":
			w.Header().Set("Content-Type", "audio/flac")
		default:
			w.Header().Set("Content-Type", "audio/octet-stream")
		}
	case models.MediaTypeVideo:
		switch ext {
		case ".mp4":
			w.Header().Set("Content-Type", "video/mp4")
		case ".webm":
			w.Header().Set("Content-Type", "video/webm")
		case ".avi":
			w.Header().Set("Content-Type", "video/x-msvideo")
		case ".mov":
			w.Header().Set("Content-Type", "video/quicktime")
		default:
			w.Header().Set("Content-Type", "video/octet-stream")
		}
	case models.MediaTypeDocs:
		switch ext {
		case ".txt":
			w.Header().Set("Content-Type", "text/plain")
		case ".md":
			w.Header().Set("Content-Type", "text/markdown")
		case ".html":
			w.Header().Set("Content-Type", "text/html")
		case ".pdf":
			w.Header().Set("Content-Type", "application/pdf")
		case ".doc":
			w.Header().Set("Content-Type", "application/msword")
		case ".docx":
			w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		default:
			w.Header().Set("Content-Type", "application/octet-stream")
		}
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
}

// HandleMediaRoutes is a router for unified media endpoints
func (h *Handler) HandleMediaRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/media/")
	pathParts := strings.Split(path, "/")

	if len(pathParts) < 2 {
		http.Error(w, "Invalid media route. Use /api/media/{type}/{action}", http.StatusBadRequest)
		return
	}

	mediaType := pathParts[0]
	action := pathParts[1]

	// Validate media type
	switch mediaType {
	case "images", "image", "audio", "video", "docs":
		// Valid types
	default:
		http.Error(w, "Invalid media type. Use 'image', 'audio', 'video', or 'docs'", http.StatusBadRequest)
		return
	}

	// Route to appropriate handler based on action
	switch action {
	case "galleries":
		if len(pathParts) == 2 {
			// GET /api/media/{type}/galleries
			h.HandleMediaGalleries(w, r)
		} else {
			// GET /api/media/{type}/galleries/{gallery}/{file?}
			h.HandleMediaGalleryContent(w, r)
		}
	case "upload":
		// POST /api/media/{type}/upload
		h.HandleUploadMedia(w, r)
	case "content":
		// GET /api/media/{type}/content/{gallery}/{filename} - for structured content (docs)
		h.HandleMediaContent(w, r)
	default:
		http.Error(w, "Invalid media action. Use 'galleries', 'upload', or 'content'", http.StatusBadRequest)
	}
}

// HandleMediaContent handles GET /api/media/{type}/content/{gallery}/{filename} requests for structured content
func (h *Handler) HandleMediaContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse URL path: /api/media/{type}/content/{gallery}/{filename}
	pathParts := strings.Split(r.URL.Path[len("/api/media/"):], "/")
	if len(pathParts) < 4 || pathParts[0] == "" || pathParts[1] != "content" || pathParts[2] == "" || pathParts[3] == "" {
		http.Error(w, "Invalid path format. Use /api/media/{type}/content/{gallery}/{filename}", http.StatusBadRequest)
		return
	}

	mediaTypeStr := pathParts[0]
	galleryName := pathParts[2]
	fileName := strings.Join(pathParts[3:], "/")

	// Currently only support docs for structured content
	if mediaTypeStr != "docs" {
		http.Error(w, "Structured content only supported for docs", http.StatusBadRequest)
		return
	}

	// Validate document file type
	if !h.isValidDocumentFile(fileName) {
		http.Error(w, "Invalid document file", http.StatusBadRequest)
		return
	}

	// Get the file path
	baseDir := h.appService.GetDirectoryService().GetDirectoryPath()
	var filePath string
	if galleryName == "root_docs" {
		filePath = h.findFileInMediaDirectory(baseDir, "docs", fileName)
	} else {
		filePath = filepath.Join(baseDir, "docs", galleryName, fileName)
	}

	if filePath == "" {
		http.Error(w, "Document file not found", http.StatusNotFound)
		return
	}

	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		http.Error(w, "Document file not found", http.StatusNotFound)
		return
	}

	// Load document with metadata using the existing directory service function
	doc, err := h.loadDocFromPath(fileName, filePath, fileInfo)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load document: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

// loadDocFromPath creates a Doc model from file path and info
func (h *Handler) loadDocFromPath(filename, filePath string, fileInfo os.FileInfo) (*models.Doc, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read doc file: %w", err)
	}

	contentStr := string(content)
	ext := strings.ToLower(filepath.Ext(filename))

	// Create title from filename (remove extension)
	title := filename
	if strings.HasSuffix(title, ext) {
		title = title[:len(title)-len(ext)]
	}

	// Generate preview (first 200 characters)
	preview := contentStr
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}

	// Determine content type and process content
	var processedContent string
	var contentType string

	if ext == ".md" {
		// For markdown files, return raw markdown content
		processedContent = contentStr
		contentType = "markdown" // Frontend will convert to HTML
	} else if ext == ".html" || ext == ".htm" {
		// For HTML files, sanitize by removing JavaScript
		processedContent = h.sanitizeHTML(contentStr)
		contentType = "html" // Frontend will open in new tab
	} else {
		processedContent = contentStr
		contentType = "text"
	}

	return &models.Doc{
		Filename:    filename,
		Title:       title,
		Content:     processedContent,
		Preview:     preview,
		ModifiedAt:  fileInfo.ModTime(),
		Size:        fileInfo.Size(),
		ContentType: contentType,
	}, nil
}

// sanitizeHTML removes all JavaScript from HTML content for security
func (h *Handler) sanitizeHTML(htmlContent string) string {
	content := htmlContent

	// Remove <script> tags and their content (case-insensitive, multiline)
	scriptRegex := `(?is)<script[^>]*>.*?</script>`
	re := regexp.MustCompile(scriptRegex)
	content = re.ReplaceAllString(content, "")

	// Remove standalone <script> tags (malformed)
	standaloneScriptRegex := `(?is)<script[^>]*/?>`
	re = regexp.MustCompile(standaloneScriptRegex)
	content = re.ReplaceAllString(content, "")

	// Remove all event handlers (on* attributes) - improved pattern
	eventHandlerRegex := `(?is)\s+on[a-zA-Z]+\s*=\s*["'][^"']*["']`
	re = regexp.MustCompile(eventHandlerRegex)
	content = re.ReplaceAllString(content, "")

	// Remove event handlers without quotes
	eventHandlerNoQuotesRegex := `(?is)\s+on[a-zA-Z]+\s*=\s*[^"'\s>][^>\s]*`
	re = regexp.MustCompile(eventHandlerNoQuotesRegex)
	content = re.ReplaceAllString(content, "")

	// Remove javascript: URLs (href="javascript:..." or src="javascript:...")
	jsUrlRegex := `(?is)(href|src|action)\s*=\s*["']javascript:[^"']*["']`
	re = regexp.MustCompile(jsUrlRegex)
	content = re.ReplaceAllString(content, `$1="#"`)

	// Remove javascript: URLs without quotes
	jsUrlNoQuotesRegex := `(?is)(href|src|action)\s*=\s*javascript:[^>\s]*`
	re = regexp.MustCompile(jsUrlNoQuotesRegex)
	content = re.ReplaceAllString(content, `$1="#"`)

	// Remove data: URLs that might contain JavaScript
	dataUrlRegex := `(?is)(href|src|action)\s*=\s*["']data:[^"']*javascript[^"']*["']`
	re = regexp.MustCompile(dataUrlRegex)
	content = re.ReplaceAllString(content, `$1="#"`)

	// Remove vbscript: URLs
	vbscriptUrlRegex := `(?is)(href|src|action)\s*=\s*["']vbscript:[^"']*["']`
	re = regexp.MustCompile(vbscriptUrlRegex)
	content = re.ReplaceAllString(content, `$1="#"`)

	// Remove <noscript> tags but keep content
	noscriptRegex := `(?is)</?noscript[^>]*>`
	re = regexp.MustCompile(noscriptRegex)
	content = re.ReplaceAllString(content, "")

	// Remove any remaining standalone javascript: references
	jsStandaloneRegex := `(?is)javascript:[^"'\s>]*`
	re = regexp.MustCompile(jsStandaloneRegex)
	content = re.ReplaceAllString(content, "#")

	return content
}

// RegisterRoutes registers all HTTP routes
func (h *Handler) RegisterRoutes() {
	// Page routes
	http.HandleFunc("/", h.HandleProfilePage)
	http.HandleFunc("/profile", h.HandleProfilePage)
	http.HandleFunc("/friends", h.HandleFriendsPage)
	http.HandleFunc("/friend-profile", h.HandleFriendProfilePage)

	// API routes
	http.HandleFunc("/api/info", h.HandleGetInfo)
	http.HandleFunc("/api/create", h.HandleCreate)
	http.HandleFunc("/api/discover", h.HandleDiscover)
	http.HandleFunc("/api/peers", h.HandlePeers)
	http.HandleFunc("/api/monitor", h.HandleMonitorStatus)
	http.HandleFunc("/api/connect-ip", h.HandleConnectByIP)
	http.HandleFunc("/api/peer-avatar/", h.HandlePeerAvatar)
	http.HandleFunc("/api/friends", h.HandleFriends)
	http.HandleFunc("/api/friends/", h.HandleFriend)
	http.HandleFunc("/api/peer-friends/", h.HandlePeerFriends)
	http.HandleFunc("/api/peer-docs/", h.HandlePeerDocs)

	// Files sync routes
	http.HandleFunc("/api/sync-friend-files", h.HandleSyncFriendFiles)

	// Peer galleries routes
	http.HandleFunc("/api/peer-galleries/", h.HandlePeerGalleries)

	// Downloaded content routes
	http.HandleFunc("/api/downloaded/", h.HandleDownloadedContent)

	// Delete routes
	http.HandleFunc("/api/delete/docs/", h.HandleDeleteDoc)
	http.HandleFunc("/api/delete/images/", h.HandleDeleteImage)

	// Subdirectory suggestion routes
	http.HandleFunc("/api/subdirectories/docs", h.HandleDocsSubdirectories)

	// Unified media routes
	http.HandleFunc("/api/media/", h.HandleMediaRoutes)
}
