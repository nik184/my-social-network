package services

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"old-school/internal/models"
	"old-school/internal/utils"
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

	// Unified media gallery methods
	GetMediaGalleries(mediaType models.MediaType) ([]models.MediaGallery, error)
	GetMediaGalleryFiles(mediaType models.MediaType, galleryName string) ([]string, error)
	GetPeerMediaGalleries(peerID string, mediaType models.MediaType) ([]models.MediaGallery, error)
	GetPeerMediaGalleryFiles(peerID, galleryName string, mediaType models.MediaType) ([]string, error)
	GetMediaGalleryNames(mediaType models.MediaType) ([]string, error)

	GetDocsSubdirectories() ([]string, error)
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

		// Only process .txt and .md files
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if ext != ".txt" && ext != ".md" {
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
	ext := strings.ToLower(filepath.Ext(filename))

	// Create title from filename (remove extension)
	title := filename
	if strings.HasSuffix(title, ext) {
		title = title[:len(title)-len(ext)]
	}

	// Handle content based on file type
	var processedContent string
	var contentType string
	var preview string

	if ext == ".md" {
		// Convert Markdown to HTML
		var buf bytes.Buffer
		if err := goldmark.Convert(content, &buf); err != nil {
			return nil, fmt.Errorf("failed to convert markdown to HTML: %w", err)
		}
		processedContent = buf.String()
		contentType = "html"

		// Create preview from original markdown (first 150 characters)
		preview = contentStr
		if len(preview) > 150 {
			preview = preview[:150] + "..."
		}
	} else {
		// Plain text file
		processedContent = contentStr
		contentType = "text"

		// Create preview (first 150 characters)
		preview = contentStr
		if len(preview) > 150 {
			preview = preview[:150] + "..."
		}
	}

	doc := &models.Doc{
		Filename:    filename,
		Title:       title,
		Content:     processedContent,
		Preview:     preview,
		ModifiedAt:  fileInfo.ModTime(),
		Size:        fileInfo.Size(),
		ContentType: contentType,
	}

	return doc, nil
}

// Unified media gallery methods

// GetMediaGalleries returns a list of all media galleries for the specified type
func (d *DirectoryService) GetMediaGalleries(mediaType models.MediaType) ([]models.MediaGallery, error) {
	var mediaDir string
	var fileCheckFunc func(string) bool
	var rootGalleryName string

	switch mediaType {
	case models.MediaTypeImage:
		mediaDir = d.pathManager.GetImagesPath()
		fileCheckFunc = utils.IsImageFile
		rootGalleryName = "root_images"
	case models.MediaTypeAudio:
		mediaDir = d.pathManager.GetAudioPath()
		fileCheckFunc = utils.IsAudioFile
		rootGalleryName = "root_audio"
	case models.MediaTypeVideo:
		mediaDir = d.pathManager.GetVideoPath()
		fileCheckFunc = utils.IsVideoFile
		rootGalleryName = "root_video"
	case models.MediaTypeDocs:
		mediaDir = d.pathManager.GetDocsPath()
		fileCheckFunc = utils.IsDocFile
		rootGalleryName = "root_docs"
	default:
		return nil, fmt.Errorf("unsupported media type: %s", mediaType)
	}

	if _, err := os.Stat(mediaDir); os.IsNotExist(err) {
		return []models.MediaGallery{}, nil
	}

	files, err := os.ReadDir(mediaDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s directory: %w", mediaType, err)
	}

	var galleries []models.MediaGallery
	var rootFiles []string

	for _, file := range files {
		if file.IsDir() {
			galleryPath := filepath.Join(mediaDir, file.Name())
			galleryFiles, err := os.ReadDir(galleryPath)
			if err != nil {
				continue
			}

			var mediaFiles []string
			for _, galleryFile := range galleryFiles {
				if galleryFile.IsDir() {
					continue
				}

				if fileCheckFunc(galleryFile.Name()) {
					mediaFiles = append(mediaFiles, galleryFile.Name())
					// Also add subdirectory files to root gallery
					rootFiles = append(rootFiles, galleryFile.Name())
				}
			}

			gallery := models.MediaGallery{
				Name:      file.Name(),
				MediaType: mediaType,
				FileCount: len(mediaFiles),
				Files:     mediaFiles,
			}

			galleries = append(galleries, gallery)
		} else {
			if fileCheckFunc(file.Name()) {
				rootFiles = append(rootFiles, file.Name())
			}
		}
	}

	if len(rootFiles) > 0 {
		rootGallery := models.MediaGallery{
			Name:      rootGalleryName,
			MediaType: mediaType,
			FileCount: len(rootFiles),
			Files:     rootFiles,
		}
		galleries = append([]models.MediaGallery{rootGallery}, galleries...)
	}

	return galleries, nil
}

// GetMediaGalleryFiles returns a list of files in a specific media gallery
func (d *DirectoryService) GetMediaGalleryFiles(mediaType models.MediaType, galleryName string) ([]string, error) {
	var mediaDir string
	var fileCheckFunc func(string) bool
	var rootGalleryName string

	switch mediaType {
	case models.MediaTypeImage:
		mediaDir = d.pathManager.GetImagesPath()
		fileCheckFunc = utils.IsImageFile
		rootGalleryName = "root_images"
	case models.MediaTypeAudio:
		mediaDir = d.pathManager.GetAudioPath()
		fileCheckFunc = utils.IsAudioFile
		rootGalleryName = "root_audio"
	case models.MediaTypeVideo:
		mediaDir = d.pathManager.GetVideoPath()
		fileCheckFunc = utils.IsVideoFile
		rootGalleryName = "root_video"
	case models.MediaTypeDocs:
		mediaDir = d.pathManager.GetDocsPath()
		fileCheckFunc = utils.IsDocFile
		rootGalleryName = "root_docs"
	default:
		return nil, fmt.Errorf("unsupported media type: %s", mediaType)
	}

	var galleryDir string
	if galleryName == rootGalleryName {
		galleryDir = mediaDir
	} else {
		if err := d.pathValidator.ValidateGalleryName(galleryName); err != nil {
			return nil, fmt.Errorf("invalid gallery name: %s - %w", galleryName, err)
		}
		galleryDir = filepath.Join(mediaDir, galleryName)
	}

	if _, err := os.Stat(galleryDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	files, err := os.ReadDir(galleryDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s gallery directory: %w", mediaType, err)
	}

	var mediaFiles []string
	for _, file := range files {
		if file.IsDir() {
			if galleryName == rootGalleryName {
				// For root galleries, also include files from subdirectories
				subDirPath := filepath.Join(galleryDir, file.Name())
				subFiles, err := os.ReadDir(subDirPath)
				if err != nil {
					continue
				}

				for _, subFile := range subFiles {
					if !subFile.IsDir() && fileCheckFunc(subFile.Name()) {
						mediaFiles = append(mediaFiles, subFile.Name())
					}
				}
			}
			continue
		}

		if fileCheckFunc(file.Name()) {
			mediaFiles = append(mediaFiles, file.Name())
		}
	}

	return mediaFiles, nil
}

// GetPeerMediaGalleries returns a list of all downloaded media galleries for a specific peer
func (d *DirectoryService) GetPeerMediaGalleries(peerID string, mediaType models.MediaType) ([]models.MediaGallery, error) {
	var peerMediaDir string
	var fileCheckFunc func(string) bool

	switch mediaType {
	case models.MediaTypeImage:
		peerMediaDir = d.pathManager.GetPeerImagesPath(peerID)
		fileCheckFunc = utils.IsImageFile
	case models.MediaTypeAudio:
		peerMediaDir = d.pathManager.GetPeerAudioPath(peerID)
		fileCheckFunc = utils.IsAudioFile
	case models.MediaTypeVideo:
		peerMediaDir = d.pathManager.GetPeerVideoPath(peerID)
		fileCheckFunc = utils.IsVideoFile
	case models.MediaTypeDocs:
		peerMediaDir = d.pathManager.GetPeerDocsPath(peerID)
		fileCheckFunc = utils.IsDocFile
	default:
		return nil, fmt.Errorf("unsupported media type: %s", mediaType)
	}

	if _, err := os.Stat(peerMediaDir); os.IsNotExist(err) {
		return []models.MediaGallery{}, nil
	}

	files, err := os.ReadDir(peerMediaDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read peer %s directory: %w", mediaType, err)
	}

	var galleries []models.MediaGallery

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		galleryPath := filepath.Join(peerMediaDir, file.Name())
		galleryFiles, err := os.ReadDir(galleryPath)
		if err != nil {
			continue
		}

		var mediaFiles []string
		for _, galleryFile := range galleryFiles {
			if galleryFile.IsDir() {
				continue
			}

			if fileCheckFunc(galleryFile.Name()) {
				mediaFiles = append(mediaFiles, galleryFile.Name())
			}
		}

		gallery := models.MediaGallery{
			Name:      file.Name(),
			MediaType: mediaType,
			FileCount: len(mediaFiles),
			Files:     mediaFiles,
		}

		galleries = append(galleries, gallery)
	}

	return galleries, nil
}

// GetPeerMediaGalleryFiles returns a list of files in a specific peer's media gallery
func (d *DirectoryService) GetPeerMediaGalleryFiles(peerID, galleryName string, mediaType models.MediaType) ([]string, error) {
	if err := d.pathValidator.ValidateGalleryName(galleryName); err != nil {
		return nil, fmt.Errorf("invalid gallery name: %s - %w", galleryName, err)
	}

	var galleryDir string
	var fileCheckFunc func(string) bool

	switch mediaType {
	case models.MediaTypeImage:
		galleryDir = d.pathManager.GetPeerGalleryPath(peerID, galleryName)
		fileCheckFunc = utils.IsImageFile
	case models.MediaTypeAudio:
		galleryDir = d.pathManager.GetPeerAudioGalleryPath(peerID, galleryName)
		fileCheckFunc = utils.IsAudioFile
	case models.MediaTypeVideo:
		galleryDir = d.pathManager.GetPeerVideoGalleryPath(peerID, galleryName)
		fileCheckFunc = utils.IsVideoFile
	case models.MediaTypeDocs:
		galleryDir = d.pathManager.GetPeerDocsGalleryPath(peerID, galleryName)
		fileCheckFunc = utils.IsDocFile
	default:
		return nil, fmt.Errorf("unsupported media type: %s", mediaType)
	}

	if _, err := os.Stat(galleryDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	files, err := os.ReadDir(galleryDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read peer %s gallery directory: %w", mediaType, err)
	}

	var mediaFiles []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if fileCheckFunc(file.Name()) {
			mediaFiles = append(mediaFiles, file.Name())
		}
	}

	return mediaFiles, nil
}

// GetMediaGalleryNames returns a list of existing media gallery names for the specified type
func (d *DirectoryService) GetMediaGalleryNames(mediaType models.MediaType) ([]string, error) {
	var mediaDir string

	switch mediaType {
	case models.MediaTypeImage:
		mediaDir = d.pathManager.GetImagesPath()
	case models.MediaTypeAudio:
		mediaDir = d.pathManager.GetAudioPath()
	case models.MediaTypeVideo:
		mediaDir = d.pathManager.GetVideoPath()
	default:
		return nil, fmt.Errorf("unsupported media type: %s", mediaType)
	}

	if _, err := os.Stat(mediaDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	files, err := os.ReadDir(mediaDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s directory: %w", mediaType, err)
	}

	var galleryNames []string
	for _, file := range files {
		if file.IsDir() {
			galleryNames = append(galleryNames, file.Name())
		}
	}

	return galleryNames, nil
}
