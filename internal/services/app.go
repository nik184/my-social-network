package services

import (
	"fmt"
	"log"

	"my-social-network/internal/interfaces"
	"my-social-network/internal/models"
)

// AppService provides high-level application operations and coordinates business logic
type AppService struct {
	container  *ServiceContainer
	folderInfo *models.FolderInfo
}

// NewAppService creates a new application service
func NewAppService() *AppService {
	// Initialize service container
	container, err := NewServiceContainer()
	if err != nil {
		log.Fatalf("Failed to create service container: %v", err)
	}

	appService := &AppService{
		container: container,
	}

	// Initialize monitor service now that AppService is available
	if err := container.InitializeMonitorService(appService); err != nil {
		log.Printf("⚠️ Warning: failed to initialize monitor service: %v", err)
	}

	// Perform startup tasks
	if err := container.PerformStartupTasks(); err != nil {
		log.Printf("⚠️ Warning: startup tasks failed: %v", err)
	}

	return appService
}

// GetServiceContainer returns the service container for advanced usage
func (a *AppService) GetServiceContainer() *ServiceContainer {
	return a.container
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
	p2pService := a.container.GetP2PService()
	response := &models.NodeInfoResponse{
		FolderInfo: a.folderInfo,
		Node:       p2pService.GetNode(),
	}

	// Add NAT status and peer information if available
	if p2pService != nil {
		response.IsPublicNode = p2pService.IsPublicNode()

		// Convert PeerInfo to PeerInfoJSON for serialization
		peerInfo := p2pService.GetConnectedPeerInfo()
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
	database := a.container.GetDatabase()
	if database == nil {
		return nil, fmt.Errorf("database service not available")
	}

	// Get connection history from database
	connections, err := database.GetConnectionHistory()
	if err != nil {
		return nil, fmt.Errorf("failed to get connection history: %w", err)
	}

	// Get currently connected peers
	currentlyConnected := make(map[string]bool)
	p2pService := a.container.GetP2PService()
	if p2pService != nil {
		connectedPeers := p2pService.GetConnectedPeers()
		for _, peerID := range connectedPeers {
			currentlyConnected[peerID.String()] = true
		}
	}

	// Convert to response format with current connection status
	var historyConnections []models.ConnectionHistoryItem
	for _, conn := range connections {
		item := models.ConnectionHistoryItem{
			PeerID:             conn.PeerID,
			PeerName:           conn.PeerName,
			Address:            conn.Address,
			LastConnected:      conn.LastConnected,
			ConnectionType:     conn.ConnectionType,
			IsValidated:        conn.IsValidated,
			CurrentlyConnected: currentlyConnected[conn.PeerID],
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
	p2pService := a.container.GetP2PService()
	if p2pService == nil {
		return nil, fmt.Errorf("P2P service not available")
	}

	return p2pService.GetSecondDegreeConnections()
}

// ConnectToSecondDegreePeer attempts to connect to a second-degree peer using hole punching
func (a *AppService) ConnectToSecondDegreePeer(targetPeerID, viaPeerID string) (*models.NodeInfoResponse, error) {
	p2pService := a.container.GetP2PService()
	if p2pService == nil {
		return nil, fmt.Errorf("P2P service not available")
	}

	return p2pService.ConnectToSecondDegreePeer(targetPeerID, viaPeerID)
}

// StartMonitoring starts the file system monitoring
func (a *AppService) StartMonitoring() error {
	return a.container.StartMonitoring()
}

// Close shuts down all services
func (a *AppService) Close() error {
	return a.container.Close()
}

// GetDirectoryService returns the directory service
func (a *AppService) GetDirectoryService() DirectoryServiceInterface {
	return a.container.GetDirectoryService()
}

// GetP2PService returns the P2P service
func (a *AppService) GetP2PService() *P2PService {
	return a.container.GetP2PService()
}

// GetMonitorService returns the monitor service
func (a *AppService) GetMonitorService() *MonitorService {
	return a.container.GetMonitorService()
}

// GetDatabaseService returns the database service
func (a *AppService) GetDatabaseService() interfaces.DatabaseService {
	return a.container.GetDatabase()
}
