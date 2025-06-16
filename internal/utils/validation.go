package utils

import (
	"path/filepath"
	"strings"
)

// PathValidator provides common path validation logic
type PathValidator struct{}

// ValidateFilename checks if a filename is safe from directory traversal
func (pv *PathValidator) ValidateFilename(filename string) error {
	if strings.Contains(filename, "..") ||
		strings.Contains(filename, "/") ||
		strings.Contains(filename, "\\") {
		return NewValidationError("filename", "contains invalid characters")
	}
	return nil
}

// ValidateGalleryName checks if a gallery name is safe
func (pv *PathValidator) ValidateGalleryName(galleryName string) error {
	if strings.Contains(galleryName, "..") ||
		strings.Contains(galleryName, "/") ||
		strings.Contains(galleryName, "\\") {
		return NewValidationError("gallery_name", "contains invalid characters")
	}
	return nil
}

// GetFileType determines file type based on extension
func GetFileType(extension string) string {
	imageExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".bmp": true, ".tiff": true, ".webp": true,
	}

	if imageExts[strings.ToLower(extension)] {
		return "image"
	}
	return "doc"
}

// IsImageFile checks if a file is an image based on extension
func IsImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".bmp":  true,
		".webp": true,
		".tiff": true,
	}
	return validExtensions[ext]
}

// IsTextFile checks if a file is a text/doc file based on extension
func IsTextFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExtensions := map[string]bool{
		".txt": true,
		".md":  true,
		".rst": true,
	}
	return validExtensions[ext]
}

// IsDocFile checks if a file is a document file based on extension
func IsDocFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExtensions := map[string]bool{
		".md":   true,
		".pdf":  true,
		".txt":  true,
		".html": true,
		".djvu": true,
		".doc":  true,
		".docx": true,
	}
	return validExtensions[ext]
}

// IsAudioFile checks if a file is an audio file based on extension
func IsAudioFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExtensions := map[string]bool{
		".mp3":  true,
		".wav":  true,
		".flac": true,
		".aac":  true,
		".ogg":  true,
		".m4a":  true,
		".wma":  true,
		".opus": true,
	}
	return validExtensions[ext]
}

// IsVideoFile checks if a file is a video file based on extension
func IsVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExtensions := map[string]bool{
		".mp4":  true,
		".avi":  true,
		".mkv":  true,
		".mov":  true,
		".wmv":  true,
		".flv":  true,
		".webm": true,
		".m4v":  true,
		".3gp":  true,
		".mpg":  true,
		".mpeg": true,
	}
	return validExtensions[ext]
}

// Singleton instance
var DefaultPathValidator = &PathValidator{}
