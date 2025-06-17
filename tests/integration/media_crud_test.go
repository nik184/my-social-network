package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// MediaGalleriesResponse represents the API response for media galleries
type MediaGalleriesResponse struct {
	MediaType string    `json:"media_type"`
	Galleries []Gallery `json:"galleries"`
	Count     int       `json:"count"`
}

// Gallery represents a media gallery
type Gallery struct {
	Name      string   `json:"name"`
	Files     []string `json:"files"`
	FileCount int      `json:"file_count"`
}

// MediaGalleryFilesResponse represents the API response for gallery files
type MediaGalleryFilesResponse struct {
	MediaType string   `json:"media_type"`
	Gallery   string   `json:"gallery"`
	Files     []string `json:"files"`
	Count     int      `json:"count"`
}

// UploadResponse represents the API response for file uploads
type UploadResponse struct {
	Success       bool     `json:"success"`
	MediaType     string   `json:"media_type"`
	UploadedFiles []string `json:"uploaded_files"`
	UploadedCount int      `json:"uploaded_count"`
	Errors        []string `json:"errors"`
	TargetDir     string   `json:"target_dir"`
}

// DeleteResponse represents the API response for file deletions
type DeleteResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Filename string `json:"filename"`
	Gallery  string `json:"gallery,omitempty"`
}

// TestMediaCRUDOperations tests create, read, and delete operations for all media types
func TestMediaCRUDOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping media CRUD test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	t.Log("🚀 Starting media CRUD operations test...")

	// Start a single node for testing
	dockerfile := getDockerfileContent()
	container, err := startMediaTestNode(ctx, dockerfile)
	require.NoError(t, err, "Failed to start test container")
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Warning: Failed to terminate container: %v", err)
		}
	}()

	// Wait for the application to initialize
	t.Log("⏳ Waiting for application to initialize...")
	time.Sleep(10 * time.Second)

	// Test all media types
	mediaTypes := []string{"image", "audio", "video", "docs"}

	for _, mediaType := range mediaTypes {
		t.Run(fmt.Sprintf("Test_%s_CRUD", mediaType), func(t *testing.T) {
			testMediaTypeCRUD(t, ctx, container, mediaType)
		})
	}
}

// testMediaTypeCRUD tests CRUD operations for a specific media type
func testMediaTypeCRUD(t *testing.T, ctx context.Context, container testcontainers.Container, mediaType string) {
	t.Logf("🎯 Testing CRUD operations for %s", mediaType)

	// Step 1: Test CREATE (upload files)
	t.Logf("📤 Testing file upload for %s", mediaType)
	uploadedFiles := testMediaUpload(t, ctx, container, mediaType)
	require.NotEmpty(t, uploadedFiles, "Should have uploaded at least one file")

	// Step 2: Test READ (list galleries and files)
	t.Logf("📋 Testing gallery listing for %s", mediaType)
	testMediaGalleryListing(t, ctx, container, mediaType, uploadedFiles)

	// Step 3: Test READ (fetch individual files)
	t.Logf("📄 Testing individual file access for %s", mediaType)
	testMediaFileAccess(t, ctx, container, mediaType, uploadedFiles)

	// Step 4: Test DELETE (remove files)
	t.Logf("🗑️ Testing file deletion for %s", mediaType)
	testMediaFileDeletion(t, ctx, container, mediaType, uploadedFiles)

	// Step 5: Verify files are gone
	t.Logf("✅ Verifying deletion for %s", mediaType)
	testMediaDeletionVerification(t, ctx, container, mediaType)
}

// testMediaUpload tests file upload functionality
func testMediaUpload(t *testing.T, ctx context.Context, container testcontainers.Container, mediaType string) []string {
	// Create test files based on media type
	testFiles := createTestFiles(t, mediaType)
	defer cleanupTestFiles(testFiles)

	// Create multipart form data
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Add files to the form
	for _, filePath := range testFiles {
		file, err := os.Open(filePath)
		require.NoError(t, err, "Should be able to open test file")
		defer file.Close()

		part, err := writer.CreateFormFile("files", filepath.Base(filePath))
		require.NoError(t, err, "Should be able to create form file")

		_, err = io.Copy(part, file)
		require.NoError(t, err, "Should be able to copy file content")
	}

	// Add subdirectory field
	err := writer.WriteField("subdirectory", "test-gallery")
	require.NoError(t, err, "Should be able to add subdirectory field")

	err = writer.Close()
	require.NoError(t, err, "Should be able to close multipart writer")

	// Make upload request
	endpoint := fmt.Sprintf("/api/media/%s/upload", mediaType)
	url, err := buildAPIURL(ctx, container, endpoint)
	require.NoError(t, err, "Should be able to build API URL")

	req, err := http.NewRequest("POST", url, &requestBody)
	require.NoError(t, err, "Should be able to create HTTP request")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err, "Upload request should succeed")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Upload should return 200 OK")

	// Parse response
	var uploadResp UploadResponse
	err = json.NewDecoder(resp.Body).Decode(&uploadResp)
	require.NoError(t, err, "Should be able to decode upload response")

	assert.True(t, uploadResp.Success, "Upload should be successful")
	assert.Equal(t, len(testFiles), uploadResp.UploadedCount, "Should upload all test files")
	assert.Empty(t, uploadResp.Errors, "Should have no upload errors")

	t.Logf("✅ Successfully uploaded %d %s files", uploadResp.UploadedCount, mediaType)
	return uploadResp.UploadedFiles
}

// testMediaGalleryListing tests gallery listing functionality
func testMediaGalleryListing(t *testing.T, ctx context.Context, container testcontainers.Container, mediaType string, expectedFiles []string) {
	// Test gallery listing
	endpoint := fmt.Sprintf("/api/media/%s/galleries", mediaType)
	url, err := buildAPIURL(ctx, container, endpoint)
	require.NoError(t, err, "Should be able to build API URL")

	resp, err := http.Get(url)
	require.NoError(t, err, "Gallery listing request should succeed")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Gallery listing should return 200 OK")

	var galleriesResp MediaGalleriesResponse
	err = json.NewDecoder(resp.Body).Decode(&galleriesResp)
	require.NoError(t, err, "Should be able to decode galleries response")

	assert.NotZero(t, galleriesResp.Count, "Should have at least one gallery")

	// Find our test gallery
	var testGallery *Gallery
	for _, gallery := range galleriesResp.Galleries {
		if gallery.Name == "test-gallery" {
			testGallery = &gallery
			break
		}
	}
	require.NotNil(t, testGallery, "Should find test-gallery")
	assert.Equal(t, len(expectedFiles), testGallery.FileCount, "Gallery should contain uploaded files")

	// Test specific gallery file listing
	galleryEndpoint := fmt.Sprintf("/api/media/%s/galleries/test-gallery", mediaType)
	galleryURL, err := buildAPIURL(ctx, container, galleryEndpoint)
	require.NoError(t, err, "Should be able to build gallery API URL")

	galleryResp, err := http.Get(galleryURL)
	require.NoError(t, err, "Gallery files request should succeed")
	defer galleryResp.Body.Close()

	assert.Equal(t, http.StatusOK, galleryResp.StatusCode, "Gallery files should return 200 OK")

	var filesResp MediaGalleryFilesResponse
	err = json.NewDecoder(galleryResp.Body).Decode(&filesResp)
	require.NoError(t, err, "Should be able to decode gallery files response")

	assert.Equal(t, len(expectedFiles), filesResp.Count, "Gallery should contain correct number of files")

	// Verify all uploaded files are present
	for _, expectedFile := range expectedFiles {
		assert.Contains(t, filesResp.Files, expectedFile, "Gallery should contain uploaded file")
	}

	t.Logf("✅ Successfully listed %d files in %s gallery", filesResp.Count, mediaType)
}

// testMediaFileAccess tests individual file access
func testMediaFileAccess(t *testing.T, ctx context.Context, container testcontainers.Container, mediaType string, uploadedFiles []string) {
	if len(uploadedFiles) == 0 {
		t.Skip("No uploaded files to test")
		return
	}

	// Test accessing the first uploaded file
	fileName := uploadedFiles[0]
	endpoint := fmt.Sprintf("/api/media/%s/galleries/test-gallery/%s", mediaType, fileName)
	url, err := buildAPIURL(ctx, container, endpoint)
	require.NoError(t, err, "Should be able to build file API URL")

	resp, err := http.Get(url)
	require.NoError(t, err, "File access request should succeed")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "File access should return 200 OK")

	// Verify content type is appropriate for media type
	contentType := resp.Header.Get("Content-Type")
	switch mediaType {
	case "image":
		assert.True(t, strings.HasPrefix(contentType, "image/"), "Image file should have image content type")
	case "audio":
		assert.True(t, strings.HasPrefix(contentType, "audio/"), "Audio file should have audio content type")
	case "video":
		assert.True(t, strings.HasPrefix(contentType, "video/"), "Video file should have video content type")
	case "docs":
		assert.True(t,
			strings.HasPrefix(contentType, "text/") ||
				strings.HasPrefix(contentType, "application/"),
			"Doc file should have text or application content type")
	}

	// Read some content to verify file is accessible
	content, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Should be able to read file content")
	assert.NotEmpty(t, content, "File should have content")

	t.Logf("✅ Successfully accessed %s file (%d bytes)", mediaType, len(content))
}

// testMediaFileDeletion tests file deletion functionality
func testMediaFileDeletion(t *testing.T, ctx context.Context, container testcontainers.Container, mediaType string, uploadedFiles []string) {
	if len(uploadedFiles) == 0 {
		t.Skip("No uploaded files to delete")
		return
	}

	// Check if deletion is supported for this media type
	if mediaType == "audio" || mediaType == "video" {
		t.Logf("⚠️ Deletion not yet implemented for %s files - skipping deletion test", mediaType)
		return
	}

	// Delete each uploaded file
	for _, fileName := range uploadedFiles {
		var endpoint string
		if mediaType == "docs" {
			// Docs delete endpoint: /api/delete/docs/{subdirectory}/{filename}
			endpoint = fmt.Sprintf("/api/delete/docs/test-gallery/%s", fileName)
		} else if mediaType == "image" {
			// Images delete endpoint: /api/delete/images/{gallery}/{filename}
			endpoint = fmt.Sprintf("/api/delete/images/test-gallery/%s", fileName)
		} else {
			t.Logf("⚠️ Delete endpoint not defined for media type: %s", mediaType)
			continue
		}

		url, err := buildAPIURL(ctx, container, endpoint)
		require.NoError(t, err, "Should be able to build delete API URL")

		req, err := http.NewRequest("DELETE", url, nil)
		require.NoError(t, err, "Should be able to create DELETE request")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err, "Delete request should succeed")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Delete should return 200 OK")

		var deleteResp DeleteResponse
		err = json.NewDecoder(resp.Body).Decode(&deleteResp)
		require.NoError(t, err, "Should be able to decode delete response")

		assert.True(t, deleteResp.Success, "Delete should be successful")
		assert.Equal(t, fileName, deleteResp.Filename, "Response should confirm correct file deleted")

		t.Logf("✅ Successfully deleted %s file: %s", mediaType, fileName)
	}
}

// testMediaDeletionVerification verifies that deleted files are no longer accessible
func testMediaDeletionVerification(t *testing.T, ctx context.Context, container testcontainers.Container, mediaType string) {
	// Skip verification if deletion is not implemented
	if mediaType == "audio" || mediaType == "video" {
		t.Logf("⚠️ Skipping deletion verification for %s (deletion not implemented)", mediaType)
		return
	}

	// Try to access the test gallery - it should either be empty or not exist
	endpoint := fmt.Sprintf("/api/media/%s/galleries/test-gallery", mediaType)
	url, err := buildAPIURL(ctx, container, endpoint)
	require.NoError(t, err, "Should be able to build API URL")

	resp, err := http.Get(url)
	require.NoError(t, err, "Gallery access request should succeed")
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Gallery still exists, check if it's empty
		var filesResp MediaGalleryFilesResponse
		err = json.NewDecoder(resp.Body).Decode(&filesResp)
		require.NoError(t, err, "Should be able to decode response")
		assert.Zero(t, filesResp.Count, "Gallery should be empty after deletion")
	} else {
		// Gallery might not exist anymore, which is also acceptable
		assert.Equal(t, http.StatusNotFound, resp.StatusCode, "Gallery should either be empty or not found")
	}

	t.Logf("✅ Verified %s files are deleted", mediaType)
}

// createTestFiles creates test files for the specified media type
func createTestFiles(t *testing.T, mediaType string) []string {
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("test_%s_*", mediaType))
	require.NoError(t, err, "Should be able to create temp directory")

	var files []string

	switch mediaType {
	case "image":
		// Create a simple PNG file (1x1 pixel)
		pngData := []byte{
			0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
			0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
			0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1 dimensions
			0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE, // bit depth, color type, etc.
			0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, 0x54, // IDAT chunk
			0x08, 0x99, 0x01, 0x01, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x02, 0x00, 0x01,
			0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82, // IEND chunk
		}
		filePath := filepath.Join(tempDir, "test_image.png")
		err := os.WriteFile(filePath, pngData, 0644)
		require.NoError(t, err, "Should be able to create test PNG file")
		files = append(files, filePath)

	case "audio":
		// Create a minimal WAV file
		wavData := []byte{
			// WAV header
			0x52, 0x49, 0x46, 0x46, // "RIFF"
			0x24, 0x00, 0x00, 0x00, // file size
			0x57, 0x41, 0x56, 0x45, // "WAVE"
			0x66, 0x6D, 0x74, 0x20, // "fmt "
			0x10, 0x00, 0x00, 0x00, // format chunk size
			0x01, 0x00, 0x01, 0x00, // audio format, channels
			0x44, 0xAC, 0x00, 0x00, // sample rate
			0x88, 0x58, 0x01, 0x00, // byte rate
			0x02, 0x00, 0x10, 0x00, // block align, bits per sample
			0x64, 0x61, 0x74, 0x61, // "data"
			0x00, 0x00, 0x00, 0x00, // data size
		}
		filePath := filepath.Join(tempDir, "test_audio.wav")
		err := os.WriteFile(filePath, wavData, 0644)
		require.NoError(t, err, "Should be able to create test WAV file")
		files = append(files, filePath)

	case "video":
		// Create a minimal MP4 file structure
		mp4Data := []byte{
			// Basic MP4 structure - ftyp box
			0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70, // box size and "ftyp"
			0x69, 0x73, 0x6F, 0x6D, 0x00, 0x00, 0x02, 0x00, // brand and version
			0x69, 0x73, 0x6F, 0x6D, 0x69, 0x73, 0x6F, 0x32, // compatible brands
			0x61, 0x76, 0x63, 0x31, 0x6D, 0x70, 0x34, 0x31,
			// Basic mdat box (empty)
			0x00, 0x00, 0x00, 0x08, 0x6D, 0x64, 0x61, 0x74,
		}
		filePath := filepath.Join(tempDir, "test_video.mp4")
		err := os.WriteFile(filePath, mp4Data, 0644)
		require.NoError(t, err, "Should be able to create test MP4 file")
		files = append(files, filePath)

	case "docs":
		// Create a simple text document
		content := "# Test Document\n\nThis is a test document for integration testing.\n\n## Features\n\n- Create\n- Read\n- Delete\n"
		filePath := filepath.Join(tempDir, "test_document.md")
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err, "Should be able to create test markdown file")
		files = append(files, filePath)
	}

	return files
}

// cleanupTestFiles removes temporary test files
func cleanupTestFiles(files []string) {
	for _, file := range files {
		dir := filepath.Dir(file)
		os.RemoveAll(dir)
	}
}

// startMediaTestNode starts a containerized node for media testing
func startMediaTestNode(ctx context.Context, dockerfile string) (testcontainers.Container, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find project root: %w", err)
	}

	dockerfilePath := filepath.Join(projectRoot, "Dockerfile.media-test")
	err = os.WriteFile(dockerfilePath, []byte(dockerfile), 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write temporary Dockerfile: %w", err)
	}

	defer func() {
		os.Remove(dockerfilePath)
	}()

	req := testcontainers.ContainerRequest{
		Name: "media-test-node",
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       projectRoot,
			Dockerfile:    "Dockerfile.media-test",
			PrintBuildLog: true,
		},
		ExposedPorts: []string{"6996/tcp", "9000/tcp"},
		WaitingFor:   wait.ForLog("Starting web server").WithStartupTimeout(120 * time.Second),
		Env: map[string]string{
			"NODE_NAME": "media-test",
		},
	}

	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}
