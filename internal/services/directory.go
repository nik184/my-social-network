package services

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"my-social-network/internal/models"
)

// DirectoryService handles directory operations
type DirectoryService struct {
	directoryPath string
}

// NewDirectoryService creates a new directory service
func NewDirectoryService() *DirectoryService {
	homeDir, _ := os.UserHomeDir()
	dirPath := filepath.Join(homeDir, "space184")

	return &DirectoryService{
		directoryPath: dirPath,
	}
}

// GetDirectoryPath returns the directory path
func (d *DirectoryService) GetDirectoryPath() string {
	return d.directoryPath
}

// CreateDirectory creates the space184 directory if it doesn't exist
func (d *DirectoryService) CreateDirectory() error {
	err := os.MkdirAll(d.directoryPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return nil
}

// ScanDirectory scans the directory and returns folder information
func (d *DirectoryService) ScanDirectory() (*models.FolderInfo, error) {
	files, err := os.ReadDir(d.directoryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var fileNames []string
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	return &models.FolderInfo{
		Path:     d.directoryPath,
		Files:    fileNames,
		LastScan: time.Now(),
	}, nil
}
