package services

import (
	"fmt"
	"log"

	"my-social-network/internal/interfaces"
	"my-social-network/internal/repository"
	"my-social-network/internal/utils"
)

// ServiceContainer manages all application services
type ServiceContainer struct {
	// Core repositories
	database interfaces.DatabaseService

	// Core services
	directoryService  DirectoryServiceInterface
	fileSystemService interfaces.FileSystemService
	templateService   *TemplateService
	friendService     *FriendService
	// portsService       *PortsService  // Commented out - not essential
	monitorService *MonitorService
	p2pService     *P2PService

	// Utilities
	pathManager *utils.PathManager
}

// NewServiceContainer creates and initializes all services
func NewServiceContainer() (*ServiceContainer, error) {
	container := &ServiceContainer{
		pathManager: utils.DefaultPathManager,
	}

	if err := container.initializeServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	return container, nil
}

// initializeServices initializes all services in the correct order
func (sc *ServiceContainer) initializeServices() error {
	// Initialize directory service first
	sc.directoryService = NewDirectoryService()

	// Initialize database
	dbPath := sc.pathManager.GetDatabasePath()
	database, err := repository.NewSQLiteRepository(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	sc.database = database

	// Initialize file system service
	sc.fileSystemService = NewFileScannerService(database)

	// Initialize utility services
	var err2 error
	sc.templateService, err2 = NewTemplateService("web/templates")
	if err2 != nil {
		return fmt.Errorf("failed to create template service: %w", err2)
	}
	// sc.portsService = NewPortsService()  // Commented out - not essential

	// Initialize P2P service
	sc.p2pService, err = NewP2PService(sc, database)
	if err != nil {
		return fmt.Errorf("failed to create P2P service: %w", err)
	}

	// Set peer ID function for file scanner after P2P service is available
	if fileScanner, ok := sc.fileSystemService.(*FileScannerService); ok {
		fileScanner.SetPeerIDFunc(func() string {
			if sc.p2pService != nil {
				return sc.p2pService.GetNode().ID.String()
			}
			return "unknown"
		})
	}

	// Initialize Friend service
	sc.friendService = NewFriendService(database, sc.p2pService)

	// Monitor service will be initialized later when AppService is available

	return nil
}

// InitializeMonitorService initializes the monitor service with an AppService
func (sc *ServiceContainer) InitializeMonitorService(appService *AppService) error {
	var err error
	sc.monitorService, err = NewMonitorService(sc.directoryService, appService)
	if err != nil {
		return fmt.Errorf("failed to create monitor service: %w", err)
	}
	return nil
}

// PerformStartupTasks executes initialization tasks
func (sc *ServiceContainer) PerformStartupTasks() error {
	log.Printf("🚀 Performing startup tasks...")

	// Clean up deleted files
	if err := sc.fileSystemService.CleanupDeletedFiles(); err != nil {
		log.Printf("⚠️ Warning: failed to cleanup deleted files: %v", err)
	}

	// Perform initial file scan
	if err := sc.fileSystemService.ScanFiles(); err != nil {
		log.Printf("⚠️ Warning: failed to perform initial file scan: %v", err)
	}

	// Attempt to reconnect to friends
	if sc.friendService != nil {
		sc.friendService.AttemptReconnectToAllFriends()

		// Sync friends' files metadata after reconnection
		log.Printf("📁 Starting friend files metadata sync...")
		if err := sc.friendService.SyncFriendFilesMetadata(); err != nil {
			log.Printf("⚠️ Warning: failed to sync friend files metadata: %v", err)
		}
	}

	log.Printf("✅ Startup tasks completed")
	return nil
}

// StartMonitoring starts the file system monitoring
func (sc *ServiceContainer) StartMonitoring() error {
	if sc.monitorService != nil {
		return sc.monitorService.Start()
	}
	return nil
}

// GetDatabase returns the database service
func (sc *ServiceContainer) GetDatabase() interfaces.DatabaseService {
	return sc.database
}

// GetDirectoryService returns the directory service
func (sc *ServiceContainer) GetDirectoryService() DirectoryServiceInterface {
	return sc.directoryService
}

// GetFileSystemService returns the file system service
func (sc *ServiceContainer) GetFileSystemService() interfaces.FileSystemService {
	return sc.fileSystemService
}

// GetP2PService returns the P2P service
func (sc *ServiceContainer) GetP2PService() *P2PService {
	return sc.p2pService
}

// GetMonitorService returns the monitor service
func (sc *ServiceContainer) GetMonitorService() *MonitorService {
	return sc.monitorService
}

// GetTemplateService returns the template service
func (sc *ServiceContainer) GetTemplateService() *TemplateService {
	return sc.templateService
}

// GetFriendService returns the friend service
func (sc *ServiceContainer) GetFriendService() *FriendService {
	return sc.friendService
}

// GetPathManager returns the path manager
func (sc *ServiceContainer) GetPathManager() *utils.PathManager {
	return sc.pathManager
}

// GetPortsService returns the ports service
// func (sc *ServiceContainer) GetPortsService() *PortsService {
//	return sc.portsService
// }

// Close shuts down all services
func (sc *ServiceContainer) Close() error {
	log.Printf("🛑 Shutting down services...")

	if sc.monitorService != nil {
		sc.monitorService.Stop()
	}

	if sc.p2pService != nil {
		if err := sc.p2pService.Close(); err != nil {
			log.Printf("Error closing P2P service: %v", err)
		}
	}

	if sc.database != nil {
		if err := sc.database.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
			return err
		}
	}

	log.Printf("✅ All services shut down successfully")
	return nil
}
