package models

import "time"

// FolderInfo represents information about a scanned directory
type FolderInfo struct {
	Path     string    `json:"path"`
	Files    []string  `json:"files"`
	LastScan time.Time `json:"lastScan"`
}

// NetworkNode represents a node in the distributed network
type NetworkNode struct {
	ID       string    `json:"id"`
	IP       string    `json:"ip"`
	Port     int       `json:"port"`
	LastSeen time.Time `json:"lastSeen"`
}

// DiscoveryRequest represents a request to discover a node
type DiscoveryRequest struct {
	IP string `json:"ip"`
}

// NodeInfoResponse represents the response containing node and folder information
type NodeInfoResponse struct {
	FolderInfo *FolderInfo  `json:"folderInfo"`
	Node       *NetworkNode `json:"node"`
}

// StatusResponse represents a generic status response
type StatusResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}