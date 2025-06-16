package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// PathManager handles common path operations
type PathManager struct {
	homeDir string
}

// NewPathManager creates a new path manager
func NewPathManager() (*PathManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	return &PathManager{homeDir: homeDir}, nil
}

// GetSpace184Path returns the space184 directory path
func (pm *PathManager) GetSpace184Path() string {
	return filepath.Join(pm.homeDir, "space184")
}

// GetDocsPath returns the docs directory path
func (pm *PathManager) GetDocsPath() string {
	return filepath.Join(pm.GetSpace184Path(), "docs")
}

// GetImagesPath returns the images directory path
func (pm *PathManager) GetImagesPath() string {
	return filepath.Join(pm.GetSpace184Path(), "images")
}

// GetAvatarPath returns the avatar directory path
func (pm *PathManager) GetAvatarPath() string {
	return filepath.Join(pm.GetImagesPath(), "avatar")
}

// GetPeerAvatarPath returns the avatar directory for a specific peer
func (pm *PathManager) GetPeerAvatarPath(peerID string) string {
	return filepath.Join(pm.GetSpace184Path(), "downloaded", peerID, "images")
}

// GetPeerDownloadPath returns the download directory for a specific peer
func (pm *PathManager) GetPeerDownloadPath(peerID string) string {
	return filepath.Join(pm.GetSpace184Path(), "downloaded", peerID)
}

// GetPeerDocsPath returns the docs directory for a specific peer
func (pm *PathManager) GetPeerDocsPath(peerID string) string {
	return filepath.Join(pm.GetSpace184Path(), "downloaded", peerID, "docs")
}

// GetPeerImagesPath returns the images directory for a specific peer
func (pm *PathManager) GetPeerImagesPath(peerID string) string {
	return filepath.Join(pm.GetSpace184Path(), "downloaded", peerID, "images")
}

// GetPeerGalleryPath returns the gallery directory for a specific peer and gallery
func (pm *PathManager) GetPeerGalleryPath(peerID, galleryName string) string {
	return filepath.Join(pm.GetSpace184Path(), "downloaded", peerID, "images", galleryName)
}

// GetAudioPath returns the audio directory path
func (pm *PathManager) GetAudioPath() string {
	return filepath.Join(pm.GetSpace184Path(), "audio")
}

// GetVideoPath returns the video directory path
func (pm *PathManager) GetVideoPath() string {
	return filepath.Join(pm.GetSpace184Path(), "video")
}

// GetPeerAudioPath returns the audio directory for a specific peer
func (pm *PathManager) GetPeerAudioPath(peerID string) string {
	return filepath.Join(pm.GetSpace184Path(), "downloaded", peerID, "audio")
}

// GetPeerVideoPath returns the video directory for a specific peer
func (pm *PathManager) GetPeerVideoPath(peerID string) string {
	return filepath.Join(pm.GetSpace184Path(), "downloaded", peerID, "video")
}

// GetPeerAudioGalleryPath returns the audio gallery directory for a specific peer and gallery
func (pm *PathManager) GetPeerAudioGalleryPath(peerID, galleryName string) string {
	return filepath.Join(pm.GetSpace184Path(), "downloaded", peerID, "audio", galleryName)
}

// GetPeerVideoGalleryPath returns the video gallery directory for a specific peer and gallery
func (pm *PathManager) GetPeerVideoGalleryPath(peerID, galleryName string) string {
	return filepath.Join(pm.GetSpace184Path(), "downloaded", peerID, "video", galleryName)
}

// GetPeerVideoGalleryPath returns the video gallery directory for a specific peer and gallery
func (pm *PathManager) GetPeerDocsGalleryPath(peerID, galleryName string) string {
	return filepath.Join(pm.GetSpace184Path(), "downloaded", peerID, "docs", galleryName)
}

// GetDatabasePath returns the database file path
func (pm *PathManager) GetDatabasePath() string {
	return filepath.Join(pm.GetSpace184Path(), "node.db")
}

// GetRelativePath computes relative path from home directory
func (pm *PathManager) GetRelativePath(absolutePath string) (string, error) {
	relPath, err := filepath.Rel(pm.GetSpace184Path(), absolutePath)
	if err != nil {
		return absolutePath, err // Fall back to absolute path
	}
	return relPath, nil
}

// EnsureDir creates a directory if it doesn't exist
func EnsureDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// DefaultPathManager singleton
var DefaultPathManager *PathManager

func init() {
	var err error
	DefaultPathManager, err = NewPathManager()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize default path manager: %v", err))
	}
}
