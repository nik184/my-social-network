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
}