# Integration Tests

This directory contains comprehensive integration tests for the distributed social network application.

## Test Categories

### 1. Node Connection Test (`node_connection_test.go`)
Tests peer-to-peer connectivity in containerized environments:
- Two isolated nodes connect via P2P
- Uses testcontainers for realistic simulation
- Tests bidirectional connection establishment
- Verifies peer recognition and validation

**What it tests:**
- Container-based P2P connectivity
- Peer identification protocol
- Bidirectional connection verification
- Network isolation and discovery

### 2. Media CRUD Test (`media_crud_test.go`)
Tests create, read, and delete operations for all media types:
- Images: PNG files via `/api/media/image/upload` and `/api/delete/images/`
- Audio: WAV files via `/api/media/audio/upload`
- Video: MP4 files via `/api/media/video/upload`  
- Docs: Markdown files via `/api/media/docs/upload` and `/api/delete/docs/`

**What it tests:**
- File upload functionality for all media types
- Gallery and file listing APIs
- Individual file access and content verification
- File deletion (where implemented)
- Proper content-type handling

## Running Tests

### Prerequisites
- Docker installed and running
- Go 1.23+
- Internet connection (for downloading base images)
- Sufficient disk space for containers and test files

### Run All Integration Tests
```bash
go test ./tests/integration/... -v
```

### Run Specific Test Suites
```bash
# Test node P2P connections only
go test ./tests/integration/ -run TestTwoIsolatedNodesConnection -v

# Test media CRUD operations only
go test ./tests/integration/ -run TestMediaCRUDOperations -v
```

### Run Tests in Short Mode (Skip Integration Tests)
```bash
go test ./tests/integration/... -short
```

## Test Coverage

### Node Connection Tests
- ✅ Isolated container node startup
- ✅ P2P connection establishment
- ✅ Bidirectional peer recognition
- ✅ Container networking integration

### Media CRUD Tests
- ✅ **CREATE**: File uploads for all media types
  - Images: PNG format with proper headers
  - Audio: WAV format with proper headers  
  - Video: MP4 format with proper headers
  - Docs: Markdown text files
- ✅ **READ**: Gallery and file access
  - Gallery listing APIs (`/api/media/{type}/galleries`)
  - File listing within galleries
  - Individual file access with content verification
  - Proper content-type response headers
- ✅ **DELETE**: File removal (where implemented)
  - Images: `/api/delete/images/{gallery}/{filename}`
  - Docs: `/api/delete/docs/{filename}`
  - Audio/Video: ⚠️ Deletion endpoints not yet implemented

## Test Environment

The tests use **testcontainers-go** to create isolated Docker environments that closely mirror production deployment:

### Container Setup
- Fresh application instance per test
- Clean filesystem and database
- Isolated network namespace
- Proper port mappings for API access
- Automatic container cleanup

### Test Workflow
1. **Setup**: Create containerized application instance
2. **Execute**: Run test operations via HTTP API
3. **Verify**: Assert expected responses and behavior
4. **Cleanup**: Remove containers and temporary files

## Test Implementation Details

### Media Type Test Files
- **Images**: 1x1 pixel PNG with proper PNG headers
- **Audio**: Minimal WAV file with correct audio headers
- **Video**: Basic MP4 container with proper structure
- **Docs**: Markdown text content for testing

### API Endpoint Coverage
```
POST /api/media/{type}/upload          # File uploads
GET  /api/media/{type}/galleries       # List galleries  
GET  /api/media/{type}/galleries/{gallery}  # List files in gallery
GET  /api/media/{type}/galleries/{gallery}/{file}  # Access individual file
DELETE /api/delete/images/{gallery}/{file}  # Delete image
DELETE /api/delete/docs/{file}              # Delete document
```

## Running the Tests

### Example Test Execution
```bash
# Run with verbose output to see detailed progress
go test ./tests/integration/ -run TestMediaCRUDOperations -v

# Expected output includes:
# 🚀 Starting media CRUD operations test...
# 🎯 Testing CRUD operations for image
# 📤 Testing file upload for image  
# ✅ Successfully uploaded 1 image files
# 📋 Testing gallery listing for image
# ✅ Successfully listed 1 files in image gallery
# ... and so on for each media type
```

### Performance Considerations
- Container startup: ~30-60 seconds
- Total test time: ~5-10 minutes for full suite
- Disk usage: ~500MB for containers and test files
- Network: Downloads base images on first run

## Notes
- Tests automatically skip deletion verification for audio/video (endpoints not implemented)
- Large log output is normal due to application initialization
- Tests are designed to be run in isolation and handle cleanup automatically
- Docker must be running and accessible to current user