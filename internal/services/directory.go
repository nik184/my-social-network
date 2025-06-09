package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"my-social-network/internal/models"
	"my-social-network/internal/utils"
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
	GetDocsDirectory() string
	CreateDocsDirectory() error
	GetDocs() ([]models.Doc, error)
	GetDoc(filename string) (*models.Doc, error)
	GetGalleries() ([]models.Gallery, error)
	GetGalleryImages(galleryName string) ([]string, error)
}

// DirectoryService handles directory operations
type DirectoryService struct {
	pathManager   *utils.PathManager
	pathValidator *utils.PathValidator
}

// NewDirectoryService creates a new directory service
func NewDirectoryService() *DirectoryService {
	return &DirectoryService{
		pathManager:   utils.DefaultPathManager,
		pathValidator: utils.DefaultPathValidator,
	}
}

// GetDirectoryPath returns the directory path
func (d *DirectoryService) GetDirectoryPath() string {
	return d.pathManager.GetSpace184Path()
}

// CreateDirectory creates the space184 directory if it doesn't exist
func (d *DirectoryService) CreateDirectory() error {
	err := utils.EnsureDir(d.pathManager.GetSpace184Path())
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Also create the docs directory as part of the standard setup
	if err := d.CreateDocsDirectory(); err != nil {
		return fmt.Errorf("failed to create docs directory: %w", err)
	}

	return nil
}

// ScanDirectory scans the directory and returns folder information
func (d *DirectoryService) ScanDirectory() (*models.FolderInfo, error) {
	space184Path := d.pathManager.GetSpace184Path()
	files, err := os.ReadDir(space184Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var fileNames []string
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	return &models.FolderInfo{
		Path:     space184Path,
		Files:    fileNames,
		LastScan: time.Now(),
	}, nil
}

// GetAvatarDirectory returns the path to the avatar images directory
func (d *DirectoryService) GetAvatarDirectory() string {
	return d.pathManager.GetAvatarPath()
}

// CreateAvatarDirectory creates the avatar images directory if it doesn't exist
func (d *DirectoryService) CreateAvatarDirectory() error {
	avatarDir := d.GetAvatarDirectory()
	err := utils.EnsureDir(avatarDir)
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

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if utils.IsImageFile(file.Name()) {
			imageFiles = append(imageFiles, file.Name())
		}
	}

	return imageFiles, nil
}

// GetPeerAvatarDirectory returns the path to a specific peer's avatar directory
func (d *DirectoryService) GetPeerAvatarDirectory(peerID string) string {
	return d.pathManager.GetPeerAvatarPath(peerID)
}

// CreatePeerAvatarDirectory creates the avatar directory for a specific peer
func (d *DirectoryService) CreatePeerAvatarDirectory(peerID string) error {
	peerAvatarDir := d.GetPeerAvatarDirectory(peerID)
	err := utils.EnsureDir(peerAvatarDir)
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
	if err := d.pathValidator.ValidateFilename(filename); err != nil {
		return fmt.Errorf("invalid filename: %s - %w", filename, err)
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

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if utils.IsImageFile(file.Name()) {
			imageFiles = append(imageFiles, file.Name())
		}
	}

	return imageFiles, nil
}

// GetDocsDirectory returns the path to the docs directory
func (d *DirectoryService) GetDocsDirectory() string {
	return d.pathManager.GetDocsPath()
}

// CreateDocsDirectory creates the docs directory if it doesn't exist
func (d *DirectoryService) CreateDocsDirectory() error {
	docsDir := d.GetDocsDirectory()
	err := utils.EnsureDir(docsDir)
	if err != nil {
		return fmt.Errorf("failed to create docs directory: %w", err)
	}
	return nil
}

// GetDocs returns a list of all docs in the docs directory
func (d *DirectoryService) GetDocs() ([]models.Doc, error) {
	docsDir := d.GetDocsDirectory()

	// Ensure the directory exists
	if err := d.CreateDocsDirectory(); err != nil {
		return nil, err
	}

	files, err := os.ReadDir(docsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read docs directory: %w", err)
	}

	var docs []models.Doc
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

		doc, err := d.loadDocFile(file.Name(), fileInfo)
		if err != nil {
			// Log error but continue with other files
			continue
		}

		docs = append(docs, *doc)
	}

	return docs, nil
}

// GetDoc returns a specific doc by filename
func (d *DirectoryService) GetDoc(filename string) (*models.Doc, error) {
	// Validate filename to prevent directory traversal
	if err := d.pathValidator.ValidateFilename(filename); err != nil {
		return nil, fmt.Errorf("invalid filename: %s - %w", filename, err)
	}

	docsDir := d.GetDocsDirectory()
	filePath := filepath.Join(docsDir, filename)

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return d.loadDocFile(filename, fileInfo)
}

// loadDocFile loads a doc from a file
func (d *DirectoryService) loadDocFile(filename string, fileInfo os.FileInfo) (*models.Doc, error) {
	docsDir := d.GetDocsDirectory()
	filePath := filepath.Join(docsDir, filename)

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read doc file: %w", err)
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

	doc := &models.Doc{
		Filename:   filename,
		Title:      title,
		Content:    contentStr,
		Preview:    preview,
		ModifiedAt: fileInfo.ModTime(),
		Size:       fileInfo.Size(),
	}

	return doc, nil
}

// GetGalleries returns a list of all photo galleries (subdirectories in space184/images/)
func (d *DirectoryService) GetGalleries() ([]models.Gallery, error) {
	imagesDir := d.pathManager.GetImagesPath()

	// Check if images directory exists
	if _, err := os.Stat(imagesDir); os.IsNotExist(err) {
		return []models.Gallery{}, nil // Return empty list if directory doesn't exist
	}

	files, err := os.ReadDir(imagesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read images directory: %w", err)
	}

	var galleries []models.Gallery

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

			if utils.IsImageFile(galleryFile.Name()) {
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
	if err := d.pathValidator.ValidateGalleryName(galleryName); err != nil {
		return nil, fmt.Errorf("invalid gallery name: %s - %w", galleryName, err)
	}

	galleryDir := filepath.Join(d.pathManager.GetImagesPath(), galleryName)

	// Check if directory exists
	if _, err := os.Stat(galleryDir); os.IsNotExist(err) {
		return []string{}, nil // Return empty list if directory doesn't exist
	}

	files, err := os.ReadDir(galleryDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read gallery directory: %w", err)
	}

	var imageFiles []string

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if utils.IsImageFile(file.Name()) {
			imageFiles = append(imageFiles, file.Name())
		}
	}

	return imageFiles, nil
}
