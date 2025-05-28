package services

import (
	"context"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// MonitorService handles file system monitoring for the space184 directory
type MonitorService struct {
	watcher           *fsnotify.Watcher
	directoryService  *DirectoryService
	appService        *AppService
	ctx               context.Context
	cancel            context.CancelFunc
	lastScanTime      time.Time
	debounceTimer     *time.Timer
	debounceDuration  time.Duration
	mu                sync.Mutex
}

// NewMonitorService creates a new file system monitor
func NewMonitorService(directoryService *DirectoryService, appService *AppService) (*MonitorService, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	monitor := &MonitorService{
		watcher:          watcher,
		directoryService: directoryService,
		appService:       appService,
		ctx:              ctx,
		cancel:           cancel,
		debounceDuration: 500 * time.Millisecond, // Debounce rapid file changes
	}

	return monitor, nil
}

// Start begins monitoring the space184 directory
func (m *MonitorService) Start() error {
	directoryPath := m.directoryService.GetDirectoryPath()
	
	// Perform initial scan
	log.Printf("📁 Starting directory monitoring for: %s", directoryPath)
	if err := m.performScan("Initial scan"); err != nil {
		log.Printf("Warning: Initial scan failed: %v", err)
	}

	// Add directory to watcher
	if err := m.watcher.Add(directoryPath); err != nil {
		return err
	}

	// Start monitoring goroutine
	go m.monitorLoop()

	log.Printf("👁️ File system monitoring started successfully")
	return nil
}

// Stop stops the monitoring service
func (m *MonitorService) Stop() {
	m.cancel()
	if m.watcher != nil {
		m.watcher.Close()
	}
	
	m.mu.Lock()
	if m.debounceTimer != nil {
		m.debounceTimer.Stop()
	}
	m.mu.Unlock()
	
	log.Printf("🛑 File system monitoring stopped")
}

// monitorLoop handles file system events
func (m *MonitorService) monitorLoop() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Monitor loop panic recovered: %v", r)
		}
	}()

	for {
		select {
		case <-m.ctx.Done():
			return
			
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}
			m.handleFileEvent(event)
			
		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("File watcher error: %v", err)
		}
	}
}

// handleFileEvent processes individual file system events
func (m *MonitorService) handleFileEvent(event fsnotify.Event) {
	// Filter out events we don't care about
	if m.shouldIgnoreEvent(event) {
		return
	}

	// Log the event for debugging
	eventType := m.getEventTypeString(event.Op)
	fileName := filepath.Base(event.Name)
	log.Printf("📂 File event: %s - %s", eventType, fileName)

	// Debounce rapid changes
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.debounceTimer != nil {
		m.debounceTimer.Stop()
	}

	m.debounceTimer = time.AfterFunc(m.debounceDuration, func() {
		if err := m.performScan("File change detected"); err != nil {
			log.Printf("Error during scheduled scan: %v", err)
		}
	})
}

// shouldIgnoreEvent filters out events we don't want to process
func (m *MonitorService) shouldIgnoreEvent(event fsnotify.Event) bool {
	fileName := filepath.Base(event.Name)
	
	// Ignore hidden files and temporary files
	if len(fileName) > 0 && fileName[0] == '.' {
		return true
	}
	
	// Ignore common temporary file patterns
	tempPatterns := []string{
		"~", ".tmp", ".swp", ".swo", "#", ".DS_Store",
	}
	
	for _, pattern := range tempPatterns {
		if len(fileName) > len(pattern) && fileName[len(fileName)-len(pattern):] == pattern {
			return true
		}
		if len(fileName) > 0 && fileName[0:1] == pattern {
			return true
		}
	}
	
	return false
}

// getEventTypeString converts fsnotify operation to readable string
func (m *MonitorService) getEventTypeString(op fsnotify.Op) string {
	switch {
	case op&fsnotify.Create != 0:
		return "Created"
	case op&fsnotify.Write != 0:
		return "Modified"
	case op&fsnotify.Remove != 0:
		return "Removed"
	case op&fsnotify.Rename != 0:
		return "Renamed"
	case op&fsnotify.Chmod != 0:
		return "Permissions changed"
	default:
		return "Unknown"
	}
}

// performScan executes a directory scan and updates the app state
func (m *MonitorService) performScan(reason string) error {
	// Prevent too frequent scans
	m.mu.Lock()
	now := time.Now()
	if now.Sub(m.lastScanTime) < 100*time.Millisecond {
		m.mu.Unlock()
		return nil // Skip scan if too recent
	}
	m.lastScanTime = now
	m.mu.Unlock()

	log.Printf("🔍 Scanning directory: %s", reason)
	
	// Perform the scan
	folderInfo, err := m.directoryService.ScanDirectory()
	if err != nil {
		return err
	}

	// Update app state
	m.appService.SetFolderInfo(folderInfo)
	
	fileCount := len(folderInfo.Files)
	if fileCount == 0 {
		log.Printf("✅ Directory scan complete: Empty directory")
	} else {
		log.Printf("✅ Directory scan complete: %d files found", fileCount)
		
		// Log first few files for debugging
		maxDisplay := 5
		if fileCount < maxDisplay {
			maxDisplay = fileCount
		}
		
		for i := 0; i < maxDisplay; i++ {
			log.Printf("   📄 %s", folderInfo.Files[i])
		}
		
		if fileCount > maxDisplay {
			log.Printf("   ... and %d more files", fileCount-maxDisplay)
		}
	}

	return nil
}

// GetLastScanTime returns the time of the last successful scan
func (m *MonitorService) GetLastScanTime() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastScanTime
}

// TriggerManualScan forces an immediate directory scan
func (m *MonitorService) TriggerManualScan() error {
	return m.performScan("Manual scan requested")
}