package services

import (
	"log"
	"path/filepath"

	"my-social-network/internal/models"
)

// AppService coordinates all application services
type AppService struct {
	DirectoryService *DirectoryService
	DatabaseService  *DatabaseService
	P2PService       *P2PService
	MonitorService   *MonitorService
	folderInfo       *models.FolderInfo
}

// NewAppService creates a new application service
func NewAppService() *AppService {
	appService := &AppService{
		DirectoryService: NewDirectoryService(),
	}
	
	// Create database path in space184 directory
	space184Path := appService.DirectoryService.GetDirectoryPath()
	dbPath := filepath.Join(space184Path, "node.db")
	
	// Initialize database service with space184 path
	dbService, err := NewDatabaseService(dbPath)
	if err != nil {
		log.Fatalf("Failed to create database service: %v", err)
	}
	appService.DatabaseService = dbService
	
	// Initialize P2P service with database
	p2pService, err := NewP2PService(appService, dbService)
	if err != nil {
		log.Fatalf("Failed to create P2P service: %v", err)
	}
	appService.P2PService = p2pService
	
	// Initialize monitoring service
	monitorService, err := NewMonitorService(appService.DirectoryService, appService)
	if err != nil {
		log.Fatalf("Failed to create monitor service: %v", err)
	}
	appService.MonitorService = monitorService
	
	return appService
}

// GetFolderInfo returns the current folder information
func (a *AppService) GetFolderInfo() *models.FolderInfo {
	return a.folderInfo
}

// SetFolderInfo sets the folder information
func (a *AppService) SetFolderInfo(info *models.FolderInfo) {
	a.folderInfo = info
}

// GetNodeInfo returns combined node and folder information
func (a *AppService) GetNodeInfo() *models.NodeInfoResponse {
	response := &models.NodeInfoResponse{
		FolderInfo: a.folderInfo,
		Node:       a.P2PService.GetNode(),
	}
	
	// Add NAT status and peer information if available
	if a.P2PService != nil {
		response.IsPublicNode = a.P2PService.IsPublicNode()
		
		// Convert PeerInfo to PeerInfoJSON for serialization
		peerInfo := a.P2PService.GetConnectedPeerInfo()
		if len(peerInfo) > 0 {
			response.ConnectedPeerInfo = make(map[string]*models.PeerInfoJSON)
			for peerID, info := range peerInfo {
				response.ConnectedPeerInfo[peerID.String()] = &models.PeerInfoJSON{
					ID:             info.ID.String(),
					Addresses:      info.Addresses,
					FirstSeen:      info.FirstSeen,
					LastSeen:       info.LastSeen,
					IsValidated:    info.IsValidated,
					ConnectionType: info.ConnectionType,
					Name:           info.Name,
				}
			}
		}
	}
	
	return response
}

// StartMonitoring starts the file system monitoring
func (a *AppService) StartMonitoring() error {
	if a.MonitorService != nil {
		return a.MonitorService.Start()
	}
	return nil
}

// Close shuts down all services
func (a *AppService) Close() error {
	if a.MonitorService != nil {
		a.MonitorService.Stop()
	}
	if a.P2PService != nil {
		if err := a.P2PService.Close(); err != nil {
			log.Printf("Error closing P2P service: %v", err)
		}
	}
	if a.DatabaseService != nil {
		return a.DatabaseService.Close()
	}
	return nil
}