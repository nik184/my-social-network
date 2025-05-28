package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"my-social-network/internal/models"
)

// NetworkService handles network operations
type NetworkService struct {
	node *models.NetworkNode
}

// NewNetworkService creates a new network service
func NewNetworkService() *NetworkService {
	return &NetworkService{
		node: &models.NetworkNode{
			ID:   generateNodeID(),
			IP:   "127.0.0.1",
			Port: 8080,
		},
	}
}

// GetNode returns the current network node
func (n *NetworkService) GetNode() *models.NetworkNode {
	return n.node
}

// DiscoverNode attempts to discover another node by IP address
func (n *NetworkService) DiscoverNode(ip string) (*models.NodeInfoResponse, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	
	resp, err := client.Get(fmt.Sprintf("http://%s:8080/api/info", ip))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to node: %w", err)
	}
	defer resp.Body.Close()
	
	var result models.NodeInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &result, nil
}

// generateNodeID generates a unique node ID
func generateNodeID() string {
	return fmt.Sprintf("node_%d", time.Now().Unix())
}