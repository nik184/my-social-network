package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"my-social-network/internal/models"
)

// DirectoryServiceInterface defines the interface for directory operations
type DirectoryServiceInterface interface {
	GetDirectoryPath() string
	CreateDirectory() error
	ScanDirectory() (*models.FolderInfo, error)
	GetAvatarDirectory() string
	CreateAvatarDirectory() error
	GetAvatarImages() ([]string, error)
}

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

// GetAvatarDirectory returns the path to the avatar images directory
func (d *DirectoryService) GetAvatarDirectory() string {
	return filepath.Join(d.directoryPath, "images", "avatar")
}

// CreateAvatarDirectory creates the avatar images directory if it doesn't exist
func (d *DirectoryService) CreateAvatarDirectory() error {
	avatarDir := d.GetAvatarDirectory()
	err := os.MkdirAll(avatarDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create avatar directory: %w", err)
	}
	return nil
}

// GetAvatarImages returns a list of image files in the avatar directory
func (d *DirectoryService) GetAvatarImages() ([]string, error) {
	avatarDir := d.GetAvatarDirectory()
	
	// Ensure the directory exists
	if err := d.CreateAvatarDirectory(); err != nil {
		return nil, err
	}

	files, err := os.ReadDir(avatarDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read avatar directory: %w", err)
	}

	var imageFiles []string
	validExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".bmp":  true,
		".webp": true,
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if validExtensions[ext] {
			imageFiles = append(imageFiles, file.Name())
		}
	}

	return imageFiles, nil
}
