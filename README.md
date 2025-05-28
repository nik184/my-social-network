# Distributed Social Network

A simple distributed Go application that manages a shared directory (`space184`) and allows network discovery between nodes.

## Features

1. **Directory Management**: Creates and manages a `space184` folder in your home directory
2. **Folder Scanning**: Scans directory contents and stores information in memory  
3. **Network Discovery**: Discovers other nodes on the network via IP address
4. **WebView UI**: Clean web-based interface for all interactions

## Project Structure

```
my-social-network/
├── cmd/
│   └── distributed-app/
│       └── main.go              # Application entry point
├── internal/
│   ├── models/
│   │   └── models.go            # Data structures
│   ├── services/
│   │   ├── app.go               # Application service coordinator
│   │   ├── directory.go         # Directory management
│   │   └── network.go           # Network operations
│   ├── handlers/
│   │   └── handlers.go          # HTTP request handlers
│   └── ui/
│       └── webview.go           # WebView interface management
├── web/
│   └── static/
│       └── index.html           # Web UI
├── go.mod                       # Go module file
├── go.sum                       # Go dependencies
└── README.md                    # This file
```

## Requirements

- Go 1.21 or later
- CGO enabled (for WebView support)
- On Linux: `sudo apt-get install webkit2gtk-4.0-dev` or similar WebKit development libraries

## Usage

1. **Build the application**:
   ```bash
   go build -o distributed-app cmd/distributed-app/main.go
   ```

2. **Run the application**:
   ```bash
   ./distributed-app
   ```

   Or run directly with:
   ```bash
   go run cmd/distributed-app/main.go
   ```

3. **Use the WebView interface** to:
   - Create the `space184` directory in your home folder
   - Scan the directory to store file information
   - Discover other nodes by entering their IP addresses
   - View current node information

## API Endpoints

The application exposes a REST API on port 8080:

- `GET /api/info` - Get current node and folder information
- `POST /api/create` - Create the space184 directory
- `POST /api/scan` - Scan the directory and update file list
- `POST /api/discover` - Discover another node (requires IP in JSON body)

## Network Discovery

To discover another node:
1. Ensure both applications are running
2. Use the "Network Discovery" section in the UI
3. Enter the IP address of the target node
4. Click "Discover Node" to retrieve their folder information

## Architecture

The application follows standard Go project layout:

- **cmd/**: Application entrypoints
- **internal/**: Private application code
  - **models/**: Data structures and types
  - **services/**: Business logic layer
  - **handlers/**: HTTP request handling
  - **ui/**: User interface management
- **web/**: Static web assets

### Components:

- **WebView UI**: Native desktop interface using webview_go
- **HTTP Server**: REST API for node communication on port 8080  
- **File System**: Direct interaction with OS file system for directory management
- **In-Memory Storage**: Folder information cached in application memory