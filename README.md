# Distributed Social Network

A truly distributed Go application using libp2p that manages a shared directory (`space184`) and enables peer-to-peer discovery and communication across the internet.

## Features

1. **Directory Management**: Creates and manages a `space184` folder in your home directory
2. **Folder Scanning**: Scans directory contents and stores information in memory  
3. **P2P Network Discovery**: Discovers peers using DHT (Distributed Hash Table) and peer IDs
4. **NAT Traversal**: Automatic hole punching and relay connections for NAT'd networks
5. **Secure Communication**: Encrypted libp2p streams for peer communication
6. **Global Connectivity**: Works across the internet without fixed IP addresses
7. **Dynamic Port Allocation**: Automatically finds available ports to avoid conflicts
8. **Real-time File Monitoring**: Automatically detects changes in the space184 directory
9. **WebView UI**: Clean web-based interface for all interactions

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
│   │   ├── monitor.go           # File system monitoring
│   │   ├── p2p.go               # libp2p P2P networking
│   │   └── ports.go             # Port allocation utilities
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

The application exposes a REST API on port 6996:

- `GET /api/info` - Get current node and folder information
- `POST /api/create` - Create the space184 directory
- `POST /api/scan` - Manually trigger directory scan
- `POST /api/discover` - Discover a peer (requires peer ID in JSON body)
- `GET /api/peers` - Get list of connected peers
- `GET /api/monitor` - Get file monitoring status and last scan time

## P2P Network Discovery

To discover another peer:
1. Ensure both applications are running
2. Use the "P2P Network Discovery" section in the UI
3. Copy your Peer ID from the "Current Node Info" section and share it
4. Enter another peer's Peer ID to discover and connect to them
5. The DHT will automatically help discover peers across the internet

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

- **libp2p Networking**: Full P2P stack with DHT, NAT traversal, hole punching
- **WebView UI**: Native desktop interface using webview_go
- **HTTP Server**: Local REST API for UI communication on port 6996
- **P2P Streams**: Secure encrypted communication between peers
- **File System**: Direct interaction with OS file system for directory management
- **In-Memory Storage**: Folder information cached in application memory

### P2P Features:

- **DHT-based Discovery**: Uses Kademlia DHT for global peer discovery
- **NAT Traversal**: Automatic hole punching and relay connections
- **Multiple Transports**: TCP, QUIC, WebRTC support
- **Secure Communication**: Noise protocol for encryption
- **Connection Management**: Automatic connection limits and cleanup
- **Bootstrap Nodes**: Connects to IPFS bootstrap nodes for initial connectivity
- **Dynamic Port Management**: Automatically finds available ports for both HTTP server and P2P communication
- **Multiple Instance Support**: Can run multiple instances on the same machine without port conflicts
- **Real-time File Monitoring**: Uses fsnotify for instant detection of file system changes
- **Automatic Scanning**: Performs initial scan on startup and monitors for changes continuously
- **Debounced Updates**: Prevents excessive scanning during rapid file operations