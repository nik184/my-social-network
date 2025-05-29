package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"my-social-network/internal/services"
	"my-social-network/internal/ui"
)

// showConnectionString displays the connection string for sharing
func showConnectionString(appService *services.AppService) {
	connectionInfo := appService.P2PService.GetConnectionInfo()
	
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📋 CONNECTION STRING FOR SHARING")
	fmt.Println(strings.Repeat("=", 60))
	
	// Show P2P addresses (extracted from local addresses)
	fmt.Printf("🔗 P2P Listening Addresses:\n")
	var p2pPort int
	var localIP string
	
	for _, addr := range connectionInfo.LocalAddresses {
		fmt.Printf("   %s\n", addr)
		// Extract the first non-localhost IP and port for manual connection
		if strings.Contains(addr, "/ip4/") && !strings.Contains(addr, "127.0.0.1") && localIP == "" {
			parts := strings.Split(addr, "/")
			if len(parts) >= 5 {
				localIP = parts[2]      // IP address
				if port, err := strconv.Atoi(parts[4]); err == nil {
					p2pPort = port       // TCP port
				}
			}
		}
	}
	
	fmt.Println(strings.Repeat("-", 60))
	
	if connectionInfo.PublicAddress != "" && connectionInfo.Port != 0 {
		connectionString := fmt.Sprintf("%s:%d:%s", 
			connectionInfo.PublicAddress, 
			connectionInfo.Port, 
			connectionInfo.PeerID)
		fmt.Printf("🌐 Auto-detected Connection String: %s\n", connectionString)
		fmt.Printf("📋 Share this with others to connect to your node\n")
		fmt.Printf("🔗 Public Address: %s:%d\n", connectionInfo.PublicAddress, connectionInfo.Port)
	} else {
		fmt.Printf("⚠️  Auto-detection failed - Node appears behind NAT\n")
		if localIP != "" && p2pPort != 0 {
			fmt.Printf("🔧 Manual Connection String (if server has public IP):\n")
			fmt.Printf("   YOUR_PUBLIC_IP:%d:%s\n", p2pPort, connectionInfo.PeerID)
			fmt.Printf("   Example: 93.183.82.171:%d:%s\n", p2pPort, connectionInfo.PeerID)
			fmt.Printf("   ⚠️  Replace YOUR_PUBLIC_IP with your actual public IP address\n")
		}
	}
	
	fmt.Printf("🆔 Peer ID: %s\n", connectionInfo.PeerID)
	fmt.Printf("🔌 P2P Port: %d (NOT the web port!)\n", p2pPort)
	fmt.Printf("📊 NAT Status: %s\n", 
		map[bool]string{true: "Public (can accept connections)", false: "Behind NAT (needs relay)"}[connectionInfo.IsPublicNode])
	
	fmt.Printf("\n💡 IMPORTANT:\n")
	fmt.Printf("   - Use P2P port (%d) for peer connections, NOT web port\n", p2pPort)
	fmt.Printf("   - Make sure port %d is open in your firewall\n", p2pPort)
	fmt.Printf("   - Web interface is on a different port (for browser access only)\n")
	
	fmt.Println(strings.Repeat("=", 60) + "\n")
}

// showNodeInfo displays current node information
func showNodeInfo(appService *services.AppService) {
	nodeInfo := appService.GetNodeInfo()
	
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("ℹ️  CURRENT NODE INFORMATION")
	fmt.Println(strings.Repeat("=", 60))
	
	// Basic node info
	if nodeInfo.Node != nil {
		fmt.Printf("🆔 Node ID: %s\n", nodeInfo.Node.ID)
		fmt.Printf("🕒 Last Seen: %s\n", nodeInfo.Node.LastSeen.Format(time.RFC3339))
		fmt.Printf("📡 Listening Addresses:\n")
		for _, addr := range nodeInfo.Node.Addresses {
			fmt.Printf("   %s\n", addr)
		}
	}
	
	// NAT status
	fmt.Printf("🌐 NAT Status: %s\n", 
		map[bool]string{true: "Public Node (can help others)", false: "Behind NAT (seeks assistance)"}[nodeInfo.IsPublicNode])
	
	// Directory info
	if nodeInfo.FolderInfo != nil {
		fmt.Printf("📁 Directory: %s\n", nodeInfo.FolderInfo.Path)
		fmt.Printf("📄 Files Count: %d\n", len(nodeInfo.FolderInfo.Files))
		fmt.Printf("🕒 Last Scan: %s\n", nodeInfo.FolderInfo.LastScan.Format(time.RFC3339))
		if len(nodeInfo.FolderInfo.Files) > 0 {
			fmt.Printf("📋 Files:\n")
			for _, file := range nodeInfo.FolderInfo.Files {
				fmt.Printf("   📄 %s\n", file)
			}
		}
	} else {
		fmt.Printf("📁 Directory: Not scanned yet\n")
	}
	
	// Connected peers info
	if nodeInfo.ConnectedPeerInfo != nil && len(nodeInfo.ConnectedPeerInfo) > 0 {
		fmt.Printf("👥 Connected Peers: %d\n", len(nodeInfo.ConnectedPeerInfo))
		fmt.Printf("🔗 Peer Details:\n")
		for peerID, info := range nodeInfo.ConnectedPeerInfo {
			shortID := peerID
			if len(peerID) > 20 {
				shortID = peerID[:20] + "..."
			}
			fmt.Printf("   🤝 %s (%s, %s)\n", 
				shortID, 
				info.ConnectionType,
				map[bool]string{true: "validated", false: "pending"}[info.IsValidated])
		}
	} else {
		fmt.Printf("👥 Connected Peers: 0\n")
	}
	
	fmt.Println(strings.Repeat("=", 60) + "\n")
}

// showHelp displays available console commands
func showHelp() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🎮 CONSOLE COMMANDS")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("Q - Show Connection String (for sharing with others)")
	fmt.Println("W - Show Current Node Info (status, peers, files)")
	fmt.Println("H - Show this help message")
	fmt.Println("Ctrl+C - Quit application")
	fmt.Println(strings.Repeat("=", 60) + "\n")
}

// startConsoleInput handles keyboard input for console interaction
func startConsoleInput(appService *services.AppService) {
	reader := bufio.NewReader(os.Stdin)
	
	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			continue
		}
		
		command := strings.ToUpper(strings.TrimSpace(input))
		
		switch command {
		case "Q":
			showConnectionString(appService)
		case "W":
			showNodeInfo(appService)
		case "H":
			showHelp()
		case "":
			// Ignore empty input
			continue
		default:
			fmt.Printf("Unknown command: %s. Press H for help.\n", command)
		}
	}
}

func main() {
	// Initialize application services
	appService := services.NewAppService()
	defer func() {
		if err := appService.Close(); err != nil {
			log.Printf("Error closing app service: %v", err)
		}
	}()
	
	// Initialize WebView UI with automatic port discovery
	webUI, err := ui.NewWebViewUI(appService, 6996)
	if err != nil {
		log.Fatalf("Failed to create WebView UI: %v", err)
	}
	
	// Start the web server
	webUI.StartServer()
	
	// Start file system monitoring
	if err := appService.StartMonitoring(); err != nil {
		log.Printf("Warning: Failed to start file monitoring: %v", err)
	}
	
	// Show initial help message
	fmt.Println("\n🎉 Distributed Social Network - Console Mode")
	showHelp()
	fmt.Println("💡 Application is running. Web interface available but console commands enabled.")
	fmt.Printf("🌐 Web UI: http://localhost:%d/index.html\n", webUI.GetPort())
	fmt.Println("⌨️  Enter commands below (Q, W, H) or Ctrl+C to quit:")
	
	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		fmt.Println("\n👋 Shutting down...")
		appService.Close()
		os.Exit(0)
	}()
	
	// Start console input handler
	go startConsoleInput(appService)
	
	// Start the WebView interface (this will return immediately in headless environments)
	webUI.ShowWebView()
	
	// Keep the application running
	select {}
}
