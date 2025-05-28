package services

import (
	"my-social-network/internal/models"
)

// AppService coordinates all application services
type AppService struct {
	DirectoryService *DirectoryService
	NetworkService   *NetworkService
	folderInfo       *models.FolderInfo
}

// NewAppService creates a new application service
func NewAppService() *AppService {
	return &AppService{
		DirectoryService: NewDirectoryService(),
		NetworkService:   NewNetworkService(),
	}
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
		Node:       a.NetworkService.GetNode(),
	}
}