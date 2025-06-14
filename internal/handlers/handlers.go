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

	// Serve the file - handle special root gallery
	var filePath string
	if galleryName == "root_images" {
		// Files are directly in the images folder
		imagesDir := filepath.Join(h.appService.GetDirectoryService().GetDirectoryPath(), "images")
		filePath = filepath.Join(imagesDir, filename)
	} else {
		// Files are in a subdirectory
		galleryDir := filepath.Join(h.appService.GetDirectoryService().GetDirectoryPath(), "images", galleryName)
		filePath = filepath.Join(galleryDir, filename)
	}

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

// HandleUploadDocs handles POST /api/upload/docs requests
func (h *Handler) HandleUploadDocs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form with 32MB max memory
	err := r.ParseMultipartForm(32 << 20)
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
	subdirectory := r.FormValue("subdirectory")
	subdirectory = strings.TrimSpace(subdirectory)

	// Validate and sanitize subdirectory path
	if subdirectory != "" {
		subdirectory = filepath.Clean(subdirectory)
		if strings.Contains(subdirectory, "..") || strings.HasPrefix(subdirectory, "/") {
			http.Error(w, "Invalid subdirectory path", http.StatusBadRequest)
			return
		}
	}

	// Get base docs directory
	baseDocsDir := filepath.Join(h.appService.GetDirectoryService().GetDirectoryPath(), "docs")

	// Create target directory
	targetDir := baseDocsDir
	if subdirectory != "" {
		targetDir = filepath.Join(baseDocsDir, subdirectory)
	}

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		http.Error(w, "Failed to create directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Process each uploaded file
	var uploadedFiles []string
	var errors []string

	for _, fileHeader := range files {
		// Validate file extension
		if !h.isValidDocumentFile(fileHeader.Filename) {
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
			os.Remove(destPath) // Clean up partial file
			continue
		}

		uploadedFiles = append(uploadedFiles, fileHeader.Filename)
	}

	// Return response
	response := map[string]interface{}{
		"success":        len(uploadedFiles) > 0,
		"uploaded_files": uploadedFiles,
		"uploaded_count": len(uploadedFiles),
		"errors":         errors,
		"target_dir":     targetDir,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleUploadPhotos handles POST /api/upload/photos requests
func (h *Handler) HandleUploadPhotos(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form with 32MB max memory
	err := r.ParseMultipartForm(32 << 20)
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
	subdirectory := r.FormValue("subdirectory")
	subdirectory = strings.TrimSpace(subdirectory)

	// Validate and sanitize subdirectory path
	if subdirectory != "" {
		subdirectory = filepath.Clean(subdirectory)
		if strings.Contains(subdirectory, "..") || strings.HasPrefix(subdirectory, "/") {
			http.Error(w, "Invalid subdirectory path", http.StatusBadRequest)
			return
		}
	}

	// Get base images directory
	baseImagesDir := filepath.Join(h.appService.GetDirectoryService().GetDirectoryPath(), "images")

	// Create target directory
	targetDir := baseImagesDir
	if subdirectory != "" {
		targetDir = filepath.Join(baseImagesDir, subdirectory)
	}

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		http.Error(w, "Failed to create directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Process each uploaded file
	var uploadedFiles []string
	var errors []string

	for _, fileHeader := range files {
		// Validate file extension
		if !h.isValidImageFile(fileHeader.Filename) {
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
			os.Remove(destPath) // Clean up partial file
			continue
		}

		uploadedFiles = append(uploadedFiles, fileHeader.Filename)
	}

	// Return response
	response := map[string]interface{}{
		"success":        len(uploadedFiles) > 0,
		"uploaded_files": uploadedFiles,
		"uploaded_count": len(uploadedFiles),
		"errors":         errors,
		"target_dir":     targetDir,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
		galleries, err := h.appService.GetDirectoryService().GetPeerGalleries(peerID)
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
		images, err := h.appService.GetDirectoryService().GetPeerGalleryImages(peerID, galleryName)
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
		images, err := h.appService.GetDirectoryService().GetPeerGalleryImages(peerID, galleryName)
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

// HandleDeleteDoc handles DELETE /api/delete/docs/{filename} requests
func (h *Handler) HandleDeleteDoc(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract filename from URL path
	filename := r.URL.Path[len("/api/delete/docs/"):]
	if filename == "" {
		http.Error(w, "Filename required", http.StatusBadRequest)
		return
	}

	// Validate filename to prevent directory traversal
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	// Get docs directory and construct file path
	docsDir := h.appService.GetDirectoryService().GetDocsDirectory()
	filePath := filepath.Join(docsDir, filename)

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
	relativePath := filepath.Join("docs", filename)
	if err := h.appService.GetDatabaseService().DeleteFileRecordByPath(relativePath); err != nil {
		log.Printf("Warning: Failed to delete file record from database: %v", err)
	}

	log.Printf("🗑️ Deleted document: %s", filename)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"message":  "Document deleted successfully",
		"filename": filename,
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

// HandleImageGalleries handles GET /api/subdirectories/images requests
func (h *Handler) HandleImageGalleries(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	galleries, err := h.appService.GetDirectoryService().GetImageGalleryNames()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get image galleries: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"galleries": galleries,
		"count":     len(galleries),
	})
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

	// Upload routes
	http.HandleFunc("/api/upload/docs", h.HandleUploadDocs)
	http.HandleFunc("/api/upload/photos", h.HandleUploadPhotos)

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
	http.HandleFunc("/api/subdirectories/images", h.HandleImageGalleries)
	http.HandleFunc("/api/subdirectories/audio", h.HandleAudioGalleryNames)
	http.HandleFunc("/api/subdirectories/video", h.HandleVideoGalleryNames)

	// Audio and video routes
	http.HandleFunc("/api/audio-galleries", h.HandleAudioGalleries)
	http.HandleFunc("/api/audio-galleries/", h.HandleAudioGallery)
	http.HandleFunc("/api/video-galleries", h.HandleVideoGalleries)
	http.HandleFunc("/api/video-galleries/", h.HandleVideoGallery)

	// Audio and video upload routes
	http.HandleFunc("/api/upload/audio", h.HandleUploadAudio)
	http.HandleFunc("/api/upload/video", h.HandleUploadVideo)
}

// HandleAudioGalleries handles GET /api/audio-galleries requests
func (h *Handler) HandleAudioGalleries(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	audioGalleries, err := h.appService.GetDirectoryService().GetAudioGalleries()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get audio galleries: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"audio_galleries": audioGalleries,
		"count":           len(audioGalleries),
	})
}

// HandleAudioGallery handles GET /api/audio-galleries/{galleryName} and /api/audio-galleries/{galleryName}/{fileName} requests
func (h *Handler) HandleAudioGallery(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/audio-galleries/"), "/")
	if len(pathParts) < 1 || pathParts[0] == "" {
		http.Error(w, "Gallery name required", http.StatusBadRequest)
		return
	}

	galleryName := pathParts[0]

	if len(pathParts) == 1 {
		// Return list of audio files in gallery
		var audioFiles []string
		var err error

		if galleryName == "root_audio" {
			// Get files directly from audio root directory
			audioDir := filepath.Join(h.appService.GetDirectoryService().GetDirectoryPath(), "audio")
			files, readErr := os.ReadDir(audioDir)
			if readErr != nil {
				http.Error(w, fmt.Sprintf("Failed to read audio directory: %v", readErr), http.StatusInternalServerError)
				return
			}

			for _, file := range files {
				if !file.IsDir() && utils.IsAudioFile(file.Name()) {
					audioFiles = append(audioFiles, file.Name())
				}
			}
		} else {
			audioFiles, err = h.appService.GetDirectoryService().GetAudioGalleryFiles(galleryName)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to get audio files: %v", err), http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"audio_files": audioFiles,
			"count":       len(audioFiles),
		})
	} else if len(pathParts) == 2 {
		// Serve specific audio file
		fileName := pathParts[1]

		if !utils.IsAudioFile(fileName) {
			http.Error(w, "Invalid audio file", http.StatusBadRequest)
			return
		}

		var audioPath string
		if galleryName == "root_audio" {
			// Files are directly in the audio folder
			audioPath = filepath.Join(h.appService.GetDirectoryService().GetDirectoryPath(), "audio", fileName)
		} else {
			// Files are in a subdirectory
			audioPath = filepath.Join(h.appService.GetDirectoryService().GetDirectoryPath(), "audio", galleryName, fileName)
		}

		http.ServeFile(w, r, audioPath)
	} else {
		http.Error(w, "Invalid path", http.StatusBadRequest)
	}
}

// HandleVideoGalleries handles GET /api/video-galleries requests
func (h *Handler) HandleVideoGalleries(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	videoGalleries, err := h.appService.GetDirectoryService().GetVideoGalleries()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get video galleries: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"video_galleries": videoGalleries,
		"count":           len(videoGalleries),
	})
}

// HandleVideoGallery handles GET /api/video-galleries/{galleryName} and /api/video-galleries/{galleryName}/{fileName} requests
func (h *Handler) HandleVideoGallery(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/video-galleries/"), "/")
	if len(pathParts) < 1 || pathParts[0] == "" {
		http.Error(w, "Gallery name required", http.StatusBadRequest)
		return
	}

	galleryName := pathParts[0]

	if len(pathParts) == 1 {
		// Return list of video files in gallery
		var videoFiles []string
		var err error

		if galleryName == "root_video" {
			// Get files directly from video root directory
			videoDir := filepath.Join(h.appService.GetDirectoryService().GetDirectoryPath(), "video")
			files, readErr := os.ReadDir(videoDir)
			if readErr != nil {
				http.Error(w, fmt.Sprintf("Failed to read video directory: %v", readErr), http.StatusInternalServerError)
				return
			}

			for _, file := range files {
				if !file.IsDir() && utils.IsVideoFile(file.Name()) {
					videoFiles = append(videoFiles, file.Name())
				}
			}
		} else {
			videoFiles, err = h.appService.GetDirectoryService().GetVideoGalleryFiles(galleryName)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to get video files: %v", err), http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"video_files": videoFiles,
			"count":       len(videoFiles),
		})
	} else if len(pathParts) == 2 {
		// Serve specific video file
		fileName := pathParts[1]

		if !utils.IsVideoFile(fileName) {
			http.Error(w, "Invalid video file", http.StatusBadRequest)
			return
		}

		var videoPath string
		if galleryName == "root_video" {
			// Files are directly in the video folder
			videoPath = filepath.Join(h.appService.GetDirectoryService().GetDirectoryPath(), "video", fileName)
		} else {
			// Files are in a subdirectory
			videoPath = filepath.Join(h.appService.GetDirectoryService().GetDirectoryPath(), "video", galleryName, fileName)
		}

		http.ServeFile(w, r, videoPath)
	} else {
		http.Error(w, "Invalid path", http.StatusBadRequest)
	}
}

// HandleUploadAudio handles POST /api/upload/audio requests
func (h *Handler) HandleUploadAudio(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(100 << 20) // 100MB max
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		http.Error(w, "No files provided", http.StatusBadRequest)
		return
	}

	subdirectory := r.FormValue("subdirectory")

	audioDir := h.appService.GetDirectoryService().GetDirectoryPath() + "/audio"
	if subdirectory != "" {
		audioDir = filepath.Join(audioDir, subdirectory)
	}

	// Ensure audio directory exists
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create audio directory: %v", err), http.StatusInternalServerError)
		return
	}

	uploadedFiles := []string{}

	for _, fileHeader := range files {
		if !utils.IsAudioFile(fileHeader.Filename) {
			continue // Skip non-audio files
		}

		file, err := fileHeader.Open()
		if err != nil {
			log.Printf("Failed to open uploaded file: %v", err)
			continue
		}
		defer file.Close()

		destPath := filepath.Join(audioDir, fileHeader.Filename)
		destFile, err := os.Create(destPath)
		if err != nil {
			log.Printf("Failed to create destination file: %v", err)
			continue
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, file)
		if err != nil {
			log.Printf("Failed to copy file: %v", err)
			continue
		}

		uploadedFiles = append(uploadedFiles, fileHeader.Filename)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Audio files uploaded successfully",
		"files":   uploadedFiles,
		"count":   len(uploadedFiles),
	})
}

// HandleUploadVideo handles POST /api/upload/video requests
func (h *Handler) HandleUploadVideo(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(500 << 20) // 500MB max for videos
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		http.Error(w, "No files provided", http.StatusBadRequest)
		return
	}

	subdirectory := r.FormValue("subdirectory")

	videoDir := h.appService.GetDirectoryService().GetDirectoryPath() + "/video"
	if subdirectory != "" {
		videoDir = filepath.Join(videoDir, subdirectory)
	}

	// Ensure video directory exists
	if err := os.MkdirAll(videoDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create video directory: %v", err), http.StatusInternalServerError)
		return
	}

	uploadedFiles := []string{}

	for _, fileHeader := range files {
		if !utils.IsVideoFile(fileHeader.Filename) {
			continue // Skip non-video files
		}

		file, err := fileHeader.Open()
		if err != nil {
			log.Printf("Failed to open uploaded file: %v", err)
			continue
		}
		defer file.Close()

		destPath := filepath.Join(videoDir, fileHeader.Filename)
		destFile, err := os.Create(destPath)
		if err != nil {
			log.Printf("Failed to create destination file: %v", err)
			continue
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, file)
		if err != nil {
			log.Printf("Failed to copy file: %v", err)
			continue
		}

		uploadedFiles = append(uploadedFiles, fileHeader.Filename)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Video files uploaded successfully",
		"files":   uploadedFiles,
		"count":   len(uploadedFiles),
	})
}

// HandleAudioGalleryNames handles GET /api/subdirectories/audio requests
func (h *Handler) HandleAudioGalleryNames(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	galleryNames, err := h.appService.GetDirectoryService().GetAudioGalleryNames()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get audio gallery names: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"galleries": galleryNames,
		"count":     len(galleryNames),
	})
}

// HandleVideoGalleryNames handles GET /api/subdirectories/video requests
func (h *Handler) HandleVideoGalleryNames(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	galleryNames, err := h.appService.GetDirectoryService().GetVideoGalleryNames()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get video gallery names: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"galleries": galleryNames,
		"count":     len(galleryNames),
	})
}
