package handlers

import (
	"encoding/json"
	"net/http"

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
	folderInfo, err := h.appService.DirectoryService.ScanDirectory()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	h.appService.SetFolderInfo(folderInfo)
	
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
	
	nodeInfo, err := h.appService.NetworkService.DiscoverNode(req.IP)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodeInfo)
}

// RegisterRoutes registers all HTTP routes
func (h *Handler) RegisterRoutes() {
	http.HandleFunc("/api/info", h.HandleGetInfo)
	http.HandleFunc("/api/scan", h.HandleScan)
	http.HandleFunc("/api/create", h.HandleCreate)
	http.HandleFunc("/api/discover", h.HandleDiscover)
}