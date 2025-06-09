package services

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"my-social-network/internal/interfaces"
	"my-social-network/internal/models"
)

// FriendService handles friend-related operations including reconnection
type FriendService struct {
	database   interfaces.DatabaseService
	p2pService *P2PService
}

// NewFriendService creates a new friend service
func NewFriendService(database interfaces.DatabaseService, p2pService *P2PService) *FriendService {
	return &FriendService{
		database:   database,
		p2pService: p2pService,
	}
}

// AttemptReconnectToAllFriends attempts to reconnect to all friends from the database
func (fs *FriendService) AttemptReconnectToAllFriends() {
	if fs.database == nil || fs.p2pService == nil {
		log.Printf("⚠️ Warning: Database or P2P service not available for friend reconnection")
		return
	}

	log.Printf("🔄 Attempting to reconnect to friends...")

	// Get friends list from database
	friends, err := fs.database.GetFriends()
	if err != nil {
		log.Printf("⚠️ Warning: Failed to get friends list: %v", err)
		return
	}

	if len(friends) == 0 {
		log.Printf("📭 No friends found to reconnect to")
		return
	}

	log.Printf("👥 Found %d friend(s) to reconnect to", len(friends))

	// Get connection history to find last known addresses
	connectionHistory, err := fs.database.GetConnectionHistory()
	if err != nil {
		log.Printf("⚠️ Warning: Failed to get connection history: %v", err)
		return
	}

	// Create a map for quick lookup of connection info
	lastConnectionMap := make(map[string]*models.ConnectionRecord)
	for i := range connectionHistory {
		record := &connectionHistory[i]
		if record.IsValidated {
			// Keep the most recent connection for each peer
			if existing, exists := lastConnectionMap[record.PeerID]; !exists || record.LastConnected.After(existing.LastConnected) {
				lastConnectionMap[record.PeerID] = record
			}
		}
	}

	// Attempt to reconnect to each friend
	successCount := 0
	for _, friend := range friends {
		if lastConnection, exists := lastConnectionMap[friend.PeerID]; exists {
			success := fs.attemptFriendReconnection(friend, lastConnection)
			if success {
				successCount++
			}
			// Add a small delay between connection attempts to avoid overwhelming the network
			time.Sleep(500 * time.Millisecond)
		} else {
			log.Printf("⚠️ No connection history found for friend %s (%s)", friend.PeerName, friend.PeerID)
		}
	}

	log.Printf("✅ Friend reconnection completed: %d/%d successful", successCount, len(friends))
}

// attemptFriendReconnection attempts to reconnect to a specific friend
func (fs *FriendService) attemptFriendReconnection(friend models.Friend, lastConnection *models.ConnectionRecord) bool {
	ip, port, err := fs.extractIPAndPort(lastConnection.Address)
	if err != nil {
		log.Printf("⚠️ Failed to extract IP/port for friend %s: %v", friend.PeerName, err)
		return false
	}

	log.Printf("🔄 Attempting to reconnect to friend %s (%s) at %s:%d", friend.PeerName, friend.PeerID, ip, port)

	// Attempt connection with a reasonable timeout
	_, err = fs.p2pService.ConnectByIP(ip, port, friend.PeerID)
	if err != nil {
		log.Printf("❌ Failed to reconnect to friend %s: %v", friend.PeerName, err)
		return false
	}

	log.Printf("✅ Successfully reconnected to friend %s", friend.PeerName)
	return true
}

// extractIPAndPort extracts IP and port from a connection address
func (fs *FriendService) extractIPAndPort(address string) (string, int, error) {
	var ip string
	var port int
	var err error

	if strings.Contains(address, "/ip4/") {
		// Parse multiaddr format: /ip4/192.168.1.100/tcp/4001
		parts := strings.Split(address, "/")
		if len(parts) >= 5 {
			ip = parts[2]
			port, err = strconv.Atoi(parts[4])
			if err != nil {
				return "", 0, fmt.Errorf("invalid port in multiaddr: %s", parts[4])
			}
		} else {
			return "", 0, fmt.Errorf("invalid multiaddr format: %s", address)
		}
	} else if strings.Contains(address, ":") {
		// Parse IP:PORT format: 192.168.1.100:4001
		parts := strings.Split(address, ":")
		if len(parts) >= 2 {
			ip = parts[0]
			port, err = strconv.Atoi(parts[1])
			if err != nil {
				return "", 0, fmt.Errorf("invalid port: %s", parts[1])
			}
		} else {
			return "", 0, fmt.Errorf("invalid IP:PORT format: %s", address)
		}
	} else {
		return "", 0, fmt.Errorf("unsupported address format: %s", address)
	}

	if ip == "" || port == 0 {
		return "", 0, fmt.Errorf("could not extract valid IP and port from address: %s", address)
	}

	return ip, port, nil
}

// ReconnectToFriend attempts to reconnect to a specific friend by peer ID
func (fs *FriendService) ReconnectToFriend(peerID string) error {
	// Get friend info
	friends, err := fs.database.GetFriends()
	if err != nil {
		return fmt.Errorf("failed to get friends list: %w", err)
	}

	var targetFriend *models.Friend
	for _, friend := range friends {
		if friend.PeerID == peerID {
			targetFriend = &friend
			break
		}
	}

	if targetFriend == nil {
		return fmt.Errorf("friend with peer ID %s not found", peerID)
	}

	// Get connection history for this friend
	connectionHistory, err := fs.database.GetConnectionHistory()
	if err != nil {
		return fmt.Errorf("failed to get connection history: %w", err)
	}

	// Find most recent validated connection for this peer
	var lastConnection *models.ConnectionRecord
	for i := range connectionHistory {
		record := &connectionHistory[i]
		if record.PeerID == peerID && record.IsValidated {
			if lastConnection == nil || record.LastConnected.After(lastConnection.LastConnected) {
				lastConnection = record
			}
		}
	}

	if lastConnection == nil {
		return fmt.Errorf("no connection history found for friend %s", targetFriend.PeerName)
	}

	// Attempt reconnection
	success := fs.attemptFriendReconnection(*targetFriend, lastConnection)
	if !success {
		return fmt.Errorf("failed to reconnect to friend %s", targetFriend.PeerName)
	}

	return nil
}

// GetFriendsConnectionStatus returns the current connection status of all friends
func (fs *FriendService) GetFriendsConnectionStatus() ([]models.Friend, error) {
	friends, err := fs.database.GetFriends()
	if err != nil {
		return nil, fmt.Errorf("failed to get friends list: %w", err)
	}

	// Get currently connected peers
	connectedPeers := make(map[string]bool)
	if fs.p2pService != nil {
		currentPeers := fs.p2pService.GetConnectedPeers()
		for _, peerID := range currentPeers {
			connectedPeers[peerID.String()] = true
		}
	}

	// Update friend status based on current connections
	for i := range friends {
		friends[i].IsOnline = connectedPeers[friends[i].PeerID]
		if friends[i].IsOnline {
			now := time.Now()
			friends[i].LastSeen = &now
		}
	}

	return friends, nil
}