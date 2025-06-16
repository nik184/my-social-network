package services

import (
	"fmt"
	"os"
)

// GetDocsSubdirectories returns a list of existing subdirectories in the docs folder
func (d *DirectoryService) GetDocsSubdirectories() ([]string, error) {
	docsDir := d.GetDocsDirectory()

	// Check if docs directory exists
	if _, err := os.Stat(docsDir); os.IsNotExist(err) {
		return []string{}, nil // Return empty list if directory doesn't exist
	}

	files, err := os.ReadDir(docsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read docs directory: %w", err)
	}

	var subdirs []string
	for _, file := range files {
		if file.IsDir() {
			subdirs = append(subdirs, file.Name())
		}
	}

	return subdirs, nil
}

