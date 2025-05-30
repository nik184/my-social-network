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
	GetPeerAvatarDirectory(peerID string) string
	CreatePeerAvatarDirectory(peerID string) error
	SavePeerAvatar(peerID string, filename string, data []byte) error
	GetPeerAvatarImages(peerID string) ([]string, error)
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

// GetPeerAvatarDirectory returns the path to a specific peer's avatar directory
func (d *DirectoryService) GetPeerAvatarDirectory(peerID string) string {
	return filepath.Join(d.directoryPath, "downloaded", peerID, "images")
}

// CreatePeerAvatarDirectory creates the avatar directory for a specific peer
func (d *DirectoryService) CreatePeerAvatarDirectory(peerID string) error {
	peerAvatarDir := d.GetPeerAvatarDirectory(peerID)
	err := os.MkdirAll(peerAvatarDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create peer avatar directory: %w", err)
	}
	return nil
}

// SavePeerAvatar saves an avatar image for a specific peer
func (d *DirectoryService) SavePeerAvatar(peerID string, filename string, data []byte) error {
	// Ensure the directory exists
	if err := d.CreatePeerAvatarDirectory(peerID); err != nil {
		return err
	}

	// Validate filename to prevent directory traversal
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return fmt.Errorf("invalid filename: %s", filename)
	}

	peerAvatarDir := d.GetPeerAvatarDirectory(peerID)
	filePath := filepath.Join(peerAvatarDir, filename)
	
	err := os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to save peer avatar: %w", err)
	}
	
	return nil
}

// GetPeerAvatarImages returns a list of image files in a peer's avatar directory
func (d *DirectoryService) GetPeerAvatarImages(peerID string) ([]string, error) {
	peerAvatarDir := d.GetPeerAvatarDirectory(peerID)
	
	// Check if directory exists
	if _, err := os.Stat(peerAvatarDir); os.IsNotExist(err) {
		return []string{}, nil // Return empty list if directory doesn't exist
	}

	files, err := os.ReadDir(peerAvatarDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read peer avatar directory: %w", err)
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
