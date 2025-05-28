package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"my-social-network/internal/models"
)

// NetworkService handles network operations
type NetworkService struct {
	node *models.NetworkNode
}

// NewNetworkService creates a new network service
func NewNetworkService() *NetworkService {
	publicIP := getPublicIP()
	return &NetworkService{
		node: &models.NetworkNode{
			ID:   generateNodeID(),
			IP:   publicIP,
			Port: 6996,
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

	resp, err := client.Get(fmt.Sprintf("http://%s:6996/api/info", ip))
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

// getPublicIP gets the public IP address from external service
func getPublicIP() string {
	// Try multiple services in case one is down
	services := []string{
		"https://api.ipify.org",
		"https://icanhazip.com",
		"https://ipecho.net/plain",
	}
	
	client := &http.Client{Timeout: 5 * time.Second}
	
	for _, service := range services {
		resp, err := client.Get(service)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		
		if resp.StatusCode == 200 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				continue
			}
			
			ip := strings.TrimSpace(string(body))
			if ip != "" {
				return ip
			}
		}
	}
	
	// Fallback to localhost if all services fail
	return "127.0.0.1"
}
