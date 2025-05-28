package services

import (
	"log"

	"my-social-network/internal/models"
)

// AppService coordinates all application services
type AppService struct {
	DirectoryService *DirectoryService
	P2PService       *P2PService
	MonitorService   *MonitorService
	folderInfo       *models.FolderInfo
}

// NewAppService creates a new application service
func NewAppService() *AppService {
	appService := &AppService{
		DirectoryService: NewDirectoryService(),
	}
	
	// Initialize P2P service
	p2pService, err := NewP2PService(appService)
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
	return &models.NodeInfoResponse{
		FolderInfo: a.folderInfo,
		Node:       a.P2PService.GetNode(),
	}
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
		return a.P2PService.Close()
	}
	return nil
}