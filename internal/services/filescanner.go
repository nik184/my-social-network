package services

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"lukechampine.com/blake3"
)

// FileScannerService handles file system scanning and hash computation
type FileScannerService struct {
	dbService *DatabaseService
}

// NewFileScannerService creates a new file scanner service
func NewFileScannerService(dbService *DatabaseService) *FileScannerService {
	return &FileScannerService{
		dbService: dbService,
	}
}

// computeFileHash computes BLAKE3 hash of a file
func computeFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hasher := blake3.New(32, nil)
	_, err = io.Copy(hasher, file)
	if err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// getFileType determines if a file is a note or image based on extension
func getFileType(extension string) string {
	imageExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".bmp": true, ".tiff": true, ".webp": true,
	}

	if imageExts[strings.ToLower(extension)] {
		return "image"
	}
	return "note"
}

// ScanFiles scans the space184/notes and space184/images directories and updates the files table
func (fs *FileScannerService) ScanFiles() error {
	log.Printf("🔍 Starting file scan...")

	// Get user home directory for proper path resolution
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Define directories to scan with full paths
	scanDirs := []string{
		filepath.Join(homeDir, "space184", "notes"),
		filepath.Join(homeDir, "space184", "images"),
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

	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
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

		// Compute relative path from user home directory
		var relPath string
		homeDir, err := os.UserHomeDir()
		if err != nil {
			relPath = path // Fall back to absolute path if home dir fails
		} else {
			relPath, err = filepath.Rel(homeDir, path)
			if err != nil {
				relPath = path // Fall back to absolute path if relative fails
			}
		}

		// Check if file already exists in database
		exists, currentHash, err := fs.dbService.fileExistsInDB(relPath)
		if err != nil {
			log.Printf("⚠️ Error checking file in database %s: %v", relPath, err)
			return nil
		}

		// Compute file hash
		hash, err := computeFileHash(path)
		if err != nil {
			log.Printf("⚠️ Error computing hash for %s: %v", relPath, err)
			return nil
		}

		// If file exists and hash hasn't changed, skip
		if exists && currentHash == hash {
			return nil
		}

		// Determine file type
		fileType := getFileType(extension)

		// Insert or update file record
		if err := fs.dbService.upsertFileRecord(relPath, hash, info.Size(), extension, fileType); err != nil {
			log.Printf("⚠️ Error upserting file record for %s: %v", relPath, err)
			return nil
		}

		if exists {
			log.Printf("📝 Updated file: %s", relPath)
		} else {
			log.Printf("📄 Added file: %s (%s)", relPath, fileType)
		}

		return nil
	})
}

// CleanupDeletedFiles removes file records for files that no longer exist on disk
func (fs *FileScannerService) CleanupDeletedFiles() error {
	files, err := fs.dbService.GetFiles()
	if err != nil {
		return fmt.Errorf("failed to get files for cleanup: %w", err)
	}

	deletedCount := 0
	for _, file := range files {
		homeDir, _ := os.UserHomeDir()
		var relPath = filepath.Join(homeDir, file.FilePath)

		if _, err := os.Stat(relPath); os.IsNotExist(err) {
			// File no longer exists, remove from database
			if err := fs.dbService.DeleteFileRecord(file.ID); err != nil {
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