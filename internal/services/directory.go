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
	GetNotesDirectory() string
	CreateNotesDirectory() error
	GetNotes() ([]models.Note, error)
	GetNote(filename string) (*models.Note, error)
	GetGalleries() ([]models.Gallery, error)
	GetGalleryImages(galleryName string) ([]string, error)
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
	
	// Also create the notes directory as part of the standard setup
	if err := d.CreateNotesDirectory(); err != nil {
		return fmt.Errorf("failed to create notes directory: %w", err)
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

// GetNotesDirectory returns the path to the notes directory
func (d *DirectoryService) GetNotesDirectory() string {
	return filepath.Join(d.directoryPath, "notes")
}

// CreateNotesDirectory creates the notes directory if it doesn't exist
func (d *DirectoryService) CreateNotesDirectory() error {
	notesDir := d.GetNotesDirectory()
	err := os.MkdirAll(notesDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create notes directory: %w", err)
	}
	return nil
}

// GetNotes returns a list of all notes in the notes directory
func (d *DirectoryService) GetNotes() ([]models.Note, error) {
	notesDir := d.GetNotesDirectory()
	
	// Ensure the directory exists
	if err := d.CreateNotesDirectory(); err != nil {
		return nil, err
	}

	files, err := os.ReadDir(notesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read notes directory: %w", err)
	}

	var notes []models.Note
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		// Only process .txt files
		if !strings.HasSuffix(strings.ToLower(file.Name()), ".txt") {
			continue
		}

		fileInfo, err := file.Info()
		if err != nil {
			// Log error but continue with other files
			continue
		}
		
		note, err := d.loadNoteFile(file.Name(), fileInfo)
		if err != nil {
			// Log error but continue with other files
			continue
		}
		
		notes = append(notes, *note)
	}

	return notes, nil
}

// GetNote returns a specific note by filename
func (d *DirectoryService) GetNote(filename string) (*models.Note, error) {
	// Validate filename to prevent directory traversal
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return nil, fmt.Errorf("invalid filename: %s", filename)
	}

	notesDir := d.GetNotesDirectory()
	filePath := filepath.Join(notesDir, filename)
	
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return d.loadNoteFile(filename, fileInfo)
}

// loadNoteFile loads a note from a file
func (d *DirectoryService) loadNoteFile(filename string, fileInfo os.FileInfo) (*models.Note, error) {
	notesDir := d.GetNotesDirectory()
	filePath := filepath.Join(notesDir, filename)
	
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read note file: %w", err)
	}

	contentStr := string(content)
	
	// Create title from filename (remove .txt extension)
	title := filename
	if strings.HasSuffix(title, ".txt") {
		title = title[:len(title)-4]
	}
	
	// Create preview (first 150 characters)
	preview := contentStr
	if len(preview) > 150 {
		preview = preview[:150] + "..."
	}

	note := &models.Note{
		Filename:   filename,
		Title:      title,
		Content:    contentStr,
		Preview:    preview,
		ModifiedAt: fileInfo.ModTime(),
		Size:       fileInfo.Size(),
	}

	return note, nil
}

// GetGalleries returns a list of all photo galleries (subdirectories in space184/images/)
func (d *DirectoryService) GetGalleries() ([]models.Gallery, error) {
	imagesDir := filepath.Join(d.directoryPath, "images")
	
	// Check if images directory exists
	if _, err := os.Stat(imagesDir); os.IsNotExist(err) {
		return []models.Gallery{}, nil // Return empty list if directory doesn't exist
	}

	files, err := os.ReadDir(imagesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read images directory: %w", err)
	}

	var galleries []models.Gallery
	validExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".bmp":  true,
		".webp": true,
	}

	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		
		galleryPath := filepath.Join(imagesDir, file.Name())
		galleryFiles, err := os.ReadDir(galleryPath)
		if err != nil {
			continue // Skip galleries we can't read
		}

		var images []string
		for _, galleryFile := range galleryFiles {
			if galleryFile.IsDir() {
				continue
			}
			
			ext := strings.ToLower(filepath.Ext(galleryFile.Name()))
			if validExtensions[ext] {
				images = append(images, galleryFile.Name())
			}
		}

		gallery := models.Gallery{
			Name:       file.Name(),
			ImageCount: len(images),
			Images:     images,
		}
		
		galleries = append(galleries, gallery)
	}

	return galleries, nil
}

// GetGalleryImages returns a list of image files in a specific gallery
func (d *DirectoryService) GetGalleryImages(galleryName string) ([]string, error) {
	// Validate gallery name to prevent directory traversal
	if strings.Contains(galleryName, "..") || strings.Contains(galleryName, "/") || strings.Contains(galleryName, "\\") {
		return nil, fmt.Errorf("invalid gallery name: %s", galleryName)
	}

	galleryDir := filepath.Join(d.directoryPath, "images", galleryName)
	
	// Check if directory exists
	if _, err := os.Stat(galleryDir); os.IsNotExist(err) {
		return []string{}, nil // Return empty list if directory doesn't exist
	}

	files, err := os.ReadDir(galleryDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read gallery directory: %w", err)
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
