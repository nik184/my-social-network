package services

import (
	"fmt"
	"log"
	"path/filepath"

	"my-social-network/internal/models"
)

// AppService coordinates all application services
type AppService struct {
	DirectoryService DirectoryServiceInterface
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
					HasAvatar:      info.HasAvatar,
				}
			}
		}
	}
	
	return response
}

// GetConnectionHistory returns connection history with current connection status
func (a *AppService) GetConnectionHistory() (*models.ConnectionHistoryResponse, error) {
	if a.DatabaseService == nil {
		return nil, fmt.Errorf("database service not available")
	}
	
	// Get connection history from database
	connections, err := a.DatabaseService.GetConnectionHistory()
	if err != nil {
		return nil, fmt.Errorf("failed to get connection history: %w", err)
	}
	
	// Get currently connected peers
	currentlyConnected := make(map[string]bool)
	if a.P2PService != nil {
		connectedPeers := a.P2PService.GetConnectedPeers()
		for _, peerID := range connectedPeers {
			currentlyConnected[peerID.String()] = true
		}
	}
	
	// Convert to response format with current connection status
	var historyConnections []models.ConnectionHistoryItem
	for _, conn := range connections {
		item := models.ConnectionHistoryItem{
			PeerID:              conn.PeerID,
			PeerName:            conn.PeerName,
			Address:             conn.Address,
			LastConnected:       conn.LastConnected,
			ConnectionType:      conn.ConnectionType,
			IsValidated:         conn.IsValidated,
			CurrentlyConnected:  currentlyConnected[conn.PeerID],
		}
		historyConnections = append(historyConnections, item)
	}
	
	return &models.ConnectionHistoryResponse{
		Connections: historyConnections,
		Count:       len(historyConnections),
	}, nil
}

// GetSecondDegreeConnections returns second-degree peer connections
func (a *AppService) GetSecondDegreeConnections() (*models.SecondDegreeConnectionsResponse, error) {
	if a.P2PService == nil {
		return nil, fmt.Errorf("P2P service not available")
	}
	
	return a.P2PService.GetSecondDegreeConnections()
}

// ConnectToSecondDegreePeer attempts to connect to a second-degree peer using hole punching
func (a *AppService) ConnectToSecondDegreePeer(targetPeerID, viaPeerID string) (*models.NodeInfoResponse, error) {
	if a.P2PService == nil {
		return nil, fmt.Errorf("P2P service not available")
	}
	
	return a.P2PService.ConnectToSecondDegreePeer(targetPeerID, viaPeerID)
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