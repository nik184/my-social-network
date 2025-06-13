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
	GetGalleries() ([]models.Gallery, error)
	GetGalleryImages(galleryName string) ([]string, error)
	GetPeerGalleries(peerID string) ([]models.Gallery, error)
	GetPeerGalleryImages(peerID, galleryName string) ([]string, error)
	GetDocsSubdirectories() ([]string, error)
	GetImageGalleryNames() ([]string, error)
	GetAudioGalleries() ([]models.AudioGallery, error)
	GetAudioGalleryFiles(galleryName string) ([]string, error)
	GetPeerAudioGalleries(peerID string) ([]models.AudioGallery, error)
	GetPeerAudioGalleryFiles(peerID, galleryName string) ([]string, error)
	GetAudioGalleryNames() ([]string, error)
	GetVideoGalleries() ([]models.VideoGallery, error)
	GetVideoGalleryFiles(galleryName string) ([]string, error)
	GetPeerVideoGalleries(peerID string) ([]models.VideoGallery, error)
	GetPeerVideoGalleryFiles(peerID, galleryName string) ([]string, error)
	GetVideoGalleryNames() ([]string, error)
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

// GetPeerGalleries returns a list of all downloaded photo galleries for a specific peer
func (d *DirectoryService) GetPeerGalleries(peerID string) ([]models.Gallery, error) {
	peerImagesDir := d.pathManager.GetPeerImagesPath(peerID)

	// Check if peer images directory exists
	if _, err := os.Stat(peerImagesDir); os.IsNotExist(err) {
		return []models.Gallery{}, nil // Return empty list if directory doesn't exist
	}

	files, err := os.ReadDir(peerImagesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read peer images directory: %w", err)
	}

	var galleries []models.Gallery

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		galleryPath := filepath.Join(peerImagesDir, file.Name())
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

// GetPeerGalleryImages returns a list of image files in a specific peer's gallery
func (d *DirectoryService) GetPeerGalleryImages(peerID, galleryName string) ([]string, error) {
	// Validate gallery name to prevent directory traversal
	if err := d.pathValidator.ValidateGalleryName(galleryName); err != nil {
		return nil, fmt.Errorf("invalid gallery name: %s - %w", galleryName, err)
	}

	galleryDir := d.pathManager.GetPeerGalleryPath(peerID, galleryName)

	// Check if directory exists
	if _, err := os.Stat(galleryDir); os.IsNotExist(err) {
		return []string{}, nil // Return empty list if directory doesn't exist
	}

	files, err := os.ReadDir(galleryDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read peer gallery directory: %w", err)
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

// GetGalleries returns a list of all photo galleries (subdirectories in space184/images/) plus root folder files
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
	var rootImages []string

	for _, file := range files {
		if file.IsDir() {
			// Handle subdirectory galleries
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
		} else {
			// Handle root folder files
			if utils.IsImageFile(file.Name()) {
				rootImages = append(rootImages, file.Name())
			}
		}
	}

	// Add root images as a special gallery if any exist
	if len(rootImages) > 0 {
		rootGallery := models.Gallery{
			Name:       "root_images",
			ImageCount: len(rootImages),
			Images:     rootImages,
		}
		// Insert at the beginning
		galleries = append([]models.Gallery{rootGallery}, galleries...)
	}

	return galleries, nil
}

// GetGalleryImages returns a list of image files in a specific gallery
func (d *DirectoryService) GetGalleryImages(galleryName string) ([]string, error) {
	var galleryDir string

	if galleryName == "root_images" {
		// Special case for root images - read directly from images folder
		galleryDir = d.pathManager.GetImagesPath()
	} else {
		// Validate gallery name to prevent directory traversal
		if err := d.pathValidator.ValidateGalleryName(galleryName); err != nil {
			return nil, fmt.Errorf("invalid gallery name: %s - %w", galleryName, err)
		}
		galleryDir = filepath.Join(d.pathManager.GetImagesPath(), galleryName)
	}

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
			// For root images, skip directories (they are handled separately as galleries)
			if galleryName == "root_images" {
				continue
			}
			continue
		}

		if utils.IsImageFile(file.Name()) {
			imageFiles = append(imageFiles, file.Name())
		}
	}

	return imageFiles, nil
}

// GetAudioGalleries returns a list of all audio galleries (subdirectories in space184/audio/) plus root folder files
func (d *DirectoryService) GetAudioGalleries() ([]models.AudioGallery, error) {
	audioDir := d.pathManager.GetAudioPath()

	// Check if audio directory exists
	if _, err := os.Stat(audioDir); os.IsNotExist(err) {
		return []models.AudioGallery{}, nil
	}

	files, err := os.ReadDir(audioDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio directory: %w", err)
	}

	var audioGalleries []models.AudioGallery
	var rootAudioFiles []string

	for _, file := range files {
		if file.IsDir() {
			// Handle subdirectory galleries
			galleryPath := filepath.Join(audioDir, file.Name())
			galleryFiles, err := os.ReadDir(galleryPath)
			if err != nil {
				continue
			}

			var audioFiles []string
			for _, galleryFile := range galleryFiles {
				if galleryFile.IsDir() {
					continue
				}

				if utils.IsAudioFile(galleryFile.Name()) {
					audioFiles = append(audioFiles, galleryFile.Name())
				}
			}

			audioGallery := models.AudioGallery{
				Name:       file.Name(),
				AudioCount: len(audioFiles),
				AudioFiles: audioFiles,
			}

			audioGalleries = append(audioGalleries, audioGallery)
		} else {
			// Handle root folder files
			if utils.IsAudioFile(file.Name()) {
				rootAudioFiles = append(rootAudioFiles, file.Name())
			}
		}
	}

	// Add root audio files as a special gallery if any exist
	if len(rootAudioFiles) > 0 {
		rootGallery := models.AudioGallery{
			Name:       "root_audio",
			AudioCount: len(rootAudioFiles),
			AudioFiles: rootAudioFiles,
		}
		// Insert at the beginning
		audioGalleries = append([]models.AudioGallery{rootGallery}, audioGalleries...)
	}

	return audioGalleries, nil
}

// GetAudioGalleryFiles returns a list of audio files in a specific gallery
func (d *DirectoryService) GetAudioGalleryFiles(galleryName string) ([]string, error) {
	var galleryDir string

	if galleryName == "root_audio" {
		// Special case for root audio - read directly from audio folder
		galleryDir = d.pathManager.GetAudioPath()
	} else {
		if err := d.pathValidator.ValidateGalleryName(galleryName); err != nil {
			return nil, fmt.Errorf("invalid gallery name: %s - %w", galleryName, err)
		}
		galleryDir = filepath.Join(d.pathManager.GetAudioPath(), galleryName)
	}

	if _, err := os.Stat(galleryDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	files, err := os.ReadDir(galleryDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio gallery directory: %w", err)
	}

	var audioFiles []string
	for _, file := range files {
		if file.IsDir() {
			// For root audio, skip directories (they are handled separately as galleries)
			if galleryName == "root_audio" {
				continue
			}
			continue
		}

		if utils.IsAudioFile(file.Name()) {
			audioFiles = append(audioFiles, file.Name())
		}
	}

	return audioFiles, nil
}

// GetPeerAudioGalleries returns a list of all downloaded audio galleries for a specific peer
func (d *DirectoryService) GetPeerAudioGalleries(peerID string) ([]models.AudioGallery, error) {
	peerAudioDir := d.pathManager.GetPeerAudioPath(peerID)

	if _, err := os.Stat(peerAudioDir); os.IsNotExist(err) {
		return []models.AudioGallery{}, nil
	}

	files, err := os.ReadDir(peerAudioDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read peer audio directory: %w", err)
	}

	var audioGalleries []models.AudioGallery

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		galleryPath := filepath.Join(peerAudioDir, file.Name())
		galleryFiles, err := os.ReadDir(galleryPath)
		if err != nil {
			continue
		}

		var audioFiles []string
		for _, galleryFile := range galleryFiles {
			if galleryFile.IsDir() {
				continue
			}

			if utils.IsAudioFile(galleryFile.Name()) {
				audioFiles = append(audioFiles, galleryFile.Name())
			}
		}

		audioGallery := models.AudioGallery{
			Name:       file.Name(),
			AudioCount: len(audioFiles),
			AudioFiles: audioFiles,
		}

		audioGalleries = append(audioGalleries, audioGallery)
	}

	return audioGalleries, nil
}

// GetPeerAudioGalleryFiles returns a list of audio files in a specific peer's gallery
func (d *DirectoryService) GetPeerAudioGalleryFiles(peerID, galleryName string) ([]string, error) {
	if err := d.pathValidator.ValidateGalleryName(galleryName); err != nil {
		return nil, fmt.Errorf("invalid gallery name: %s - %w", galleryName, err)
	}

	galleryDir := d.pathManager.GetPeerAudioGalleryPath(peerID, galleryName)

	if _, err := os.Stat(galleryDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	files, err := os.ReadDir(galleryDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read peer audio gallery directory: %w", err)
	}

	var audioFiles []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if utils.IsAudioFile(file.Name()) {
			audioFiles = append(audioFiles, file.Name())
		}
	}

	return audioFiles, nil
}

// GetAudioGalleryNames returns a list of existing audio gallery names
func (d *DirectoryService) GetAudioGalleryNames() ([]string, error) {
	audioDir := d.pathManager.GetAudioPath()

	if _, err := os.Stat(audioDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	files, err := os.ReadDir(audioDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio directory: %w", err)
	}

	var galleryNames []string
	for _, file := range files {
		if file.IsDir() {
			galleryNames = append(galleryNames, file.Name())
		}
	}

	return galleryNames, nil
}

// Video methods...

// GetVideoGalleries returns a list of all video galleries (subdirectories in space184/video/) plus root folder files
func (d *DirectoryService) GetVideoGalleries() ([]models.VideoGallery, error) {
	videoDir := d.pathManager.GetVideoPath()

	if _, err := os.Stat(videoDir); os.IsNotExist(err) {
		return []models.VideoGallery{}, nil
	}

	files, err := os.ReadDir(videoDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read video directory: %w", err)
	}

	var videoGalleries []models.VideoGallery
	var rootVideoFiles []string

	for _, file := range files {
		if file.IsDir() {
			// Handle subdirectory galleries
			galleryPath := filepath.Join(videoDir, file.Name())
			galleryFiles, err := os.ReadDir(galleryPath)
			if err != nil {
				continue
			}

			var videoFiles []string
			for _, galleryFile := range galleryFiles {
				if galleryFile.IsDir() {
					continue
				}

				if utils.IsVideoFile(galleryFile.Name()) {
					videoFiles = append(videoFiles, galleryFile.Name())
				}
			}

			videoGallery := models.VideoGallery{
				Name:       file.Name(),
				VideoCount: len(videoFiles),
				VideoFiles: videoFiles,
			}

			videoGalleries = append(videoGalleries, videoGallery)
		} else {
			// Handle root folder files
			if utils.IsVideoFile(file.Name()) {
				rootVideoFiles = append(rootVideoFiles, file.Name())
			}
		}
	}

	// Add root video files as a special gallery if any exist
	if len(rootVideoFiles) > 0 {
		rootGallery := models.VideoGallery{
			Name:       "root_video",
			VideoCount: len(rootVideoFiles),
			VideoFiles: rootVideoFiles,
		}
		// Insert at the beginning
		videoGalleries = append([]models.VideoGallery{rootGallery}, videoGalleries...)
	}

	return videoGalleries, nil
}

// GetVideoGalleryFiles returns a list of video files in a specific gallery
func (d *DirectoryService) GetVideoGalleryFiles(galleryName string) ([]string, error) {
	var galleryDir string

	if galleryName == "root_video" {
		// Special case for root video - read directly from video folder
		galleryDir = d.pathManager.GetVideoPath()
	} else {
		if err := d.pathValidator.ValidateGalleryName(galleryName); err != nil {
			return nil, fmt.Errorf("invalid gallery name: %s - %w", galleryName, err)
		}
		galleryDir = filepath.Join(d.pathManager.GetVideoPath(), galleryName)
	}

	if _, err := os.Stat(galleryDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	files, err := os.ReadDir(galleryDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read video gallery directory: %w", err)
	}

	var videoFiles []string
	for _, file := range files {
		if file.IsDir() {
			// For root video, skip directories (they are handled separately as galleries)
			if galleryName == "root_video" {
				continue
			}
			continue
		}

		if utils.IsVideoFile(file.Name()) {
			videoFiles = append(videoFiles, file.Name())
		}
	}

	return videoFiles, nil
}

// GetPeerVideoGalleries returns a list of all downloaded video galleries for a specific peer
func (d *DirectoryService) GetPeerVideoGalleries(peerID string) ([]models.VideoGallery, error) {
	peerVideoDir := d.pathManager.GetPeerVideoPath(peerID)

	if _, err := os.Stat(peerVideoDir); os.IsNotExist(err) {
		return []models.VideoGallery{}, nil
	}

	files, err := os.ReadDir(peerVideoDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read peer video directory: %w", err)
	}

	var videoGalleries []models.VideoGallery

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		galleryPath := filepath.Join(peerVideoDir, file.Name())
		galleryFiles, err := os.ReadDir(galleryPath)
		if err != nil {
			continue
		}

		var videoFiles []string
		for _, galleryFile := range galleryFiles {
			if galleryFile.IsDir() {
				continue
			}

			if utils.IsVideoFile(galleryFile.Name()) {
				videoFiles = append(videoFiles, galleryFile.Name())
			}
		}

		videoGallery := models.VideoGallery{
			Name:       file.Name(),
			VideoCount: len(videoFiles),
			VideoFiles: videoFiles,
		}

		videoGalleries = append(videoGalleries, videoGallery)
	}

	return videoGalleries, nil
}

// GetPeerVideoGalleryFiles returns a list of video files in a specific peer's gallery
func (d *DirectoryService) GetPeerVideoGalleryFiles(peerID, galleryName string) ([]string, error) {
	if err := d.pathValidator.ValidateGalleryName(galleryName); err != nil {
		return nil, fmt.Errorf("invalid gallery name: %s - %w", galleryName, err)
	}

	galleryDir := d.pathManager.GetPeerVideoGalleryPath(peerID, galleryName)

	if _, err := os.Stat(galleryDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	files, err := os.ReadDir(galleryDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read peer video gallery directory: %w", err)
	}

	var videoFiles []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if utils.IsVideoFile(file.Name()) {
			videoFiles = append(videoFiles, file.Name())
		}
	}

	return videoFiles, nil
}

// GetVideoGalleryNames returns a list of existing video gallery names
func (d *DirectoryService) GetVideoGalleryNames() ([]string, error) {
	videoDir := d.pathManager.GetVideoPath()

	if _, err := os.Stat(videoDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	files, err := os.ReadDir(videoDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read video directory: %w", err)
	}

	var galleryNames []string
	for _, file := range files {
		if file.IsDir() {
			galleryNames = append(galleryNames, file.Name())
		}
	}

	return galleryNames, nil
}
