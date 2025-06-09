package interfaces

import (
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"my-social-network/internal/models"
)

// Repository interfaces following Repository pattern
type SettingsRepository interface {
	GetSetting(key string) (string, error)
	SetSetting(key, value string) error
	GetAllSettings() (map[string]string, error)
	GetNodePrivateKey() (crypto.PrivKey, error)
	GetNodeID() (peer.ID, error)
}

type ConnectionRepository interface {
	RecordConnection(peerID, address, connectionType string, isValidated bool) error
	RecordConnectionWithName(peerID, address, connectionType string, isValidated bool, peerName string) error
	GetConnectionHistory() ([]models.ConnectionRecord, error)
	GetRecentConnections(days int) ([]models.ConnectionRecord, error)
}

type FriendsRepository interface {
	AddFriend(peerID, peerName string) error
	RemoveFriend(peerID string) error
	GetFriends() ([]models.Friend, error)
	IsFriend(peerID string) (bool, error)
	UpdateFriendStatus(peerID string, isOnline bool) error
}

type FilesRepository interface {
	FileExists(filePath string) (bool, string, error)
	UpsertFileRecord(filePath, hash string, size int64, extension, fileType string) error
	GetFiles() ([]models.FileRecord, error)
	DeleteFileRecord(fileID int) error
}

// Service interfaces for better abstraction
type DatabaseService interface {
	SettingsRepository
	ConnectionRepository
	FriendsRepository
	FilesRepository
	Close() error
}

type FileSystemService interface {
	ScanFiles() error
	CleanupDeletedFiles() error
}

type PeerValidationService interface {
	ValidatePeer(peerID peer.ID) (bool, error)
	GetPeerValidationStatus(peerID peer.ID) bool
}

type AvatarService interface {
	ExchangeAvatars(peerID peer.ID) error
	GetPeerAvatars(peerID peer.ID) ([]string, error)
	SavePeerAvatar(peerID peer.ID, filename string, data []byte) error
}

type MessageHandler interface {
	Handle(payload interface{}, peerID peer.ID) error
	MessageType() string
}

type MessageRouter interface {
	RegisterHandler(messageType string, handler MessageHandler)
	RouteMessage(messageType string, payload interface{}, peerID peer.ID) error
}

type NetworkService interface {
	GetConnectedPeers() []peer.ID
	ConnectToPeer(address string) error
	IsPublicNode() bool
	GetNodeID() peer.ID
}

// Configuration interfaces
type NodeConfiguration interface {
	GetNodeID() (peer.ID, error)
	GetNodeName() (string, error)
	GetPrivateKey() (interface{}, error)
}

// Event interfaces for decoupling
type EventPublisher interface {
	Publish(eventType string, data interface{}) error
}

type EventSubscriber interface {
	Subscribe(eventType string, handler func(data interface{})) error
}
