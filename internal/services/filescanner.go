package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"old-school/internal/interfaces"
	"old-school/internal/utils"
)

// FileScannerService handles file system scanning and hash computation
type FileScannerService struct {
	filesRepo     interfaces.FilesRepository
	hashService   *utils.HashService
	pathManager   *utils.PathManager
	getPeerIDFunc func() string
}

// NewFileScannerService creates a new file scanner service
func NewFileScannerService(filesRepo interfaces.FilesRepository) *FileScannerService {
	return &FileScannerService{
		filesRepo:     filesRepo,
		hashService:   utils.DefaultHashService,
		pathManager:   utils.DefaultPathManager,
		getPeerIDFunc: func() string { return "unknown" }, // Default placeholder
	}
}

// SetPeerIDFunc sets the function to get current peer ID
func (fs *FileScannerService) SetPeerIDFunc(fn func() string) {
	fs.getPeerIDFunc = fn
}

// ScanFiles scans the space184/docs and space184/images directories and updates the files table
func (fs *FileScannerService) ScanFiles() error {
	log.Printf("🔍 Starting file scan...")

	// Define directories to scan using path manager
	scanDirs := []string{
		fs.pathManager.GetDocsPath(),
		fs.pathManager.GetImagesPath(),
	}

	for _, dir := range scanDirs {
		if err := fs.scanDirectory(dir); err != nil {
			log.Printf("⚠️ Warning: failed to scan directory %s: %v", dir, err)
			// Continue scanning other directories even if one fails
		}
	}

	log.Printf("✅ File scan completed")
	return nil
}

// scanDirectory scans a specific directory for files
func (fs *FileScannerService) scanDirectory(dirPath string) error {
	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		log.Printf("📁 Directory %s does not exist, skipping", dirPath)
		return nil
	}

	return filepath.Walk(dirPath, fs.processFile)
}

// processFile processes a single file during directory walking
func (fs *FileScannerService) processFile(path string, info os.FileInfo, err error) error {
	if err != nil {
		log.Printf("⚠️ Error accessing path %s: %v", path, err)
		return nil // Continue walking even if there's an error with one file
	}

	// Skip directories
	if info.IsDir() {
		return nil
	}

	// Get file extension
	extension := strings.ToLower(filepath.Ext(path))
	if extension == "" {
		return nil // Skip files without extensions
	}

	// Compute relative path using path manager
	relPath, err := fs.pathManager.GetRelativePath(path)
	if err != nil {
		log.Printf("⚠️ Error computing relative path for %s: %v", path, err)
		relPath = path // Fall back to absolute path
	}

	// Check if file already exists in database
	exists, currentHash, err := fs.filesRepo.FileExists(relPath)
	if err != nil {
		log.Printf("⚠️ Error checking file in database %s: %v", relPath, err)
		return nil
	}

	// Compute file hash using hash service
	hash, err := fs.hashService.ComputeFileHash(path)
	if err != nil {
		log.Printf("⚠️ Error computing hash for %s: %v", relPath, err)
		return nil
	}

	// If file exists and hash hasn't changed, skip
	if exists && currentHash == hash {
		return nil
	}

	// Determine file type using utility
	fileType := utils.GetFileType(extension)

	// Get current peer ID
	peerID := fs.getPeerIDFunc()

	// Insert or update file record
	if err := fs.filesRepo.UpsertFileRecord(relPath, hash, info.Size(), extension, fileType, peerID); err != nil {
		log.Printf("⚠️ Error upserting file record for %s: %v", relPath, err)
		return nil
	}

	if exists {
		log.Printf("📝 Updated file: %s", relPath)
	} else {
		log.Printf("📄 Added file: %s (%s)", relPath, fileType)
	}

	return nil
}

// CleanupDeletedFiles removes file records for files that no longer exist on disk
func (fs *FileScannerService) CleanupDeletedFiles() error {
	files, err := fs.filesRepo.GetFiles()
	if err != nil {
		return fmt.Errorf("failed to get files for cleanup: %w", err)
	}

	deletedCount := 0
	for _, file := range files {
		// Construct full path using path manager
		fullPath := filepath.Join(fs.pathManager.GetSpace184Path(), file.FilePath)

		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			// File no longer exists, remove from database
			if err := fs.filesRepo.DeleteFileRecord(file.ID); err != nil {
				log.Printf("⚠️ Failed to delete file record for %s: %v", file.FilePath, err)
				continue
			}
			log.Printf("🗑️ Removed deleted file: %s", file.FilePath)
			deletedCount++
		}
	}

	if deletedCount > 0 {
		log.Printf("🧹 Cleaned up %d deleted file records", deletedCount)
	}

	return nil
}

// Ensure FileScannerService implements the FileSystemService interface
var _ interfaces.FileSystemService = (*FileScannerService)(nil)
