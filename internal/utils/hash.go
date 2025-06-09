package utils

import (
	"fmt"
	"io"
	"os"

	"lukechampine.com/blake3"
)

// HashService provides file hashing utilities
type HashService struct{}

// ComputeFileHash computes BLAKE3 hash of a file
func (h *HashService) ComputeFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	hasher := blake3.New(32, nil)
	_, err = io.Copy(hasher, file)
	if err != nil {
		return "", fmt.Errorf("failed to hash file %s: %w", filePath, err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// ComputeDataHash computes BLAKE3 hash of byte data
func (h *HashService) ComputeDataHash(data []byte) string {
	hasher := blake3.New(32, nil)
	hasher.Write(data)
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// Singleton instance
var DefaultHashService = &HashService{}