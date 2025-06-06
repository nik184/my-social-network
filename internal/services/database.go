package services

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"lukechampine.com/blake3"
	_ "modernc.org/sqlite"

	"my-social-network/internal/models"
)

// DatabaseService handles SQLite database operations
type DatabaseService struct {
	db *sql.DB
}

// ConnectionRecord represents a connection history record
type ConnectionRecord struct {
	ID             int       `json:"id"`
	PeerID         string    `json:"peer_id"`
	Address        string    `json:"address"`
	FirstConnected time.Time `json:"first_connected"`
	LastConnected  time.Time `json:"last_connected"`
	ConnectionType string    `json:"connection_type"`
	IsValidated    bool      `json:"is_validated"`
	PeerName       string    `json:"peer_name"`
}

// FileRecord represents a file metadata record
type FileRecord struct {
	ID        int       `json:"id"`
	FilePath  string    `json:"filepath"`
	Hash      string    `json:"hash"`
	Size      int64     `json:"size"`
	Extension string    `json:"extension"`
	Type      string    `json:"type"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ensureDir creates a directory if it doesn't exist
func ensureDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// NewDatabaseService creates a new database service
func NewDatabaseService(dbPath string) (*DatabaseService, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := ensureDir(dir); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	service := &DatabaseService{db: db}

	if err := service.initTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	if err := service.migrateTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate tables: %w", err)
	}

	if err := service.initDefaultSettings(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize default settings: %w", err)
	}

	return service, nil
}

// initTables creates the required tables if they don't exist
func (d *DatabaseService) initTables() error {
	// Create settings table
	settingsTable := `
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
	`
	if _, err := d.db.Exec(settingsTable); err != nil {
		return fmt.Errorf("failed to create settings table: %w", err)
	}

	// Create connections table with peer_id uniqueness
	connectionsTable := `
		CREATE TABLE IF NOT EXISTS connections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			peer_id TEXT NOT NULL UNIQUE,
			address TEXT NOT NULL,
			first_connected DATETIME NOT NULL,
			last_connected DATETIME NOT NULL,
			connection_type TEXT NOT NULL,
			is_validated BOOLEAN NOT NULL DEFAULT 0,
			peer_name TEXT DEFAULT ''
		);
	`
	if _, err := d.db.Exec(connectionsTable); err != nil {
		return fmt.Errorf("failed to create connections table: %w", err)
	}

	// Create friends table
	friendsTable := `
		CREATE TABLE IF NOT EXISTS friends (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			peer_id TEXT NOT NULL UNIQUE,
			peer_name TEXT NOT NULL,
			added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_seen DATETIME,
			is_online BOOLEAN NOT NULL DEFAULT 0
		);
	`
	if _, err := d.db.Exec(friendsTable); err != nil {
		return fmt.Errorf("failed to create friends table: %w", err)
	}

	// Create files table
	filesTable := `
		CREATE TABLE IF NOT EXISTS files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filepath TEXT NOT NULL UNIQUE,
			hash TEXT NOT NULL,
			size INTEGER NOT NULL,
			extension TEXT NOT NULL,
			type TEXT NOT NULL,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`
	if _, err := d.db.Exec(filesTable); err != nil {
		return fmt.Errorf("failed to create files table: %w", err)
	}

	log.Printf("📊 Database tables initialized successfully")
	return nil
}

// migrateTables handles database schema migrations
func (d *DatabaseService) migrateTables() error {
	// Check if peer_name column exists in connections table
	rows, err := d.db.Query("PRAGMA table_info(connections)")
	if err != nil {
		return fmt.Errorf("failed to get table info: %w", err)
	}
	defer rows.Close()

	var hasNameColumn bool
	for rows.Next() {
		var cid int
		var name, dataType, notNull, defaultValue, pk interface{}
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("failed to scan column info: %w", err)
		}
		if nameStr, ok := name.(string); ok && nameStr == "peer_name" {
			hasNameColumn = true
			break
		}
	}

	// Add peer_name column if it doesn't exist
	if !hasNameColumn {
		_, err := d.db.Exec("ALTER TABLE connections ADD COLUMN peer_name TEXT DEFAULT ''")
		if err != nil {
			return fmt.Errorf("failed to add peer_name column: %w", err)
		}
		log.Printf("📈 Database migrated: added peer_name column to connections table")
	}

	return nil
}

// initDefaultSettings creates default settings if they don't exist
func (d *DatabaseService) initDefaultSettings() error {
	// Check if settings already exist
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM settings").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check settings count: %w", err)
	}

	if count > 0 {
		log.Printf("📋 Using existing settings from database")
		return nil
	}

	// Generate a new node private key if none exists
	nodePrivKey, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate node key: %w", err)
	}

	nodeID, err := peer.IDFromPrivateKey(nodePrivKey)
	if err != nil {
		return fmt.Errorf("failed to generate node ID: %w", err)
	}

	// Serialize the private key
	privKeyBytes, err := crypto.MarshalPrivateKey(nodePrivKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Insert default settings
	settings := map[string]string{
		"name":        "jerry",
		"node_id":     nodeID.String(),
		"private_key": string(privKeyBytes), // Store as base64-encoded string
	}

	for key, value := range settings {
		_, err := d.db.Exec("INSERT INTO settings (key, value) VALUES (?, ?)", key, value)
		if err != nil {
			return fmt.Errorf("failed to insert setting %s: %w", key, err)
		}
	}

	log.Printf("🎯 Created default settings: name=jerry, node_id=%s", nodeID.String())
	return nil
}

// GetSetting retrieves a setting value by key
func (d *DatabaseService) GetSetting(key string) (string, error) {
	var value string
	err := d.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("setting '%s' not found", key)
		}
		return "", fmt.Errorf("failed to get setting '%s': %w", key, err)
	}
	return value, nil
}

// SetSetting updates or inserts a setting
func (d *DatabaseService) SetSetting(key, value string) error {
	_, err := d.db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value)
	if err != nil {
		return fmt.Errorf("failed to set setting '%s': %w", key, err)
	}
	return nil
}

// GetAllSettings retrieves all settings as a map
func (d *DatabaseService) GetAllSettings() (map[string]string, error) {
	rows, err := d.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, fmt.Errorf("failed to query settings: %w", err)
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}
		settings[key] = value
	}

	return settings, nil
}

// GetNodePrivateKey retrieves the node's private key from the database
func (d *DatabaseService) GetNodePrivateKey() (crypto.PrivKey, error) {
	privKeyStr, err := d.GetSetting("private_key")
	if err != nil {
		return nil, fmt.Errorf("failed to get private key setting: %w", err)
	}

	privKey, err := crypto.UnmarshalPrivateKey([]byte(privKeyStr))
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal private key: %w", err)
	}

	return privKey, nil
}

// GetNodeID retrieves the node ID from the database
func (d *DatabaseService) GetNodeID() (peer.ID, error) {
	nodeIDStr, err := d.GetSetting("node_id")
	if err != nil {
		return "", fmt.Errorf("failed to get node ID setting: %w", err)
	}

	nodeID, err := peer.Decode(nodeIDStr)
	if err != nil {
		return "", fmt.Errorf("failed to decode node ID: %w", err)
	}

	return nodeID, nil
}

// RecordConnection stores or updates a connection record
func (d *DatabaseService) RecordConnection(peerID, address, connectionType string, isValidated bool) error {
	return d.RecordConnectionWithName(peerID, address, connectionType, isValidated, "")
}

// RecordConnectionWithName stores or updates a connection record with peer name
func (d *DatabaseService) RecordConnectionWithName(peerID, address, connectionType string, isValidated bool, peerName string) error {
	now := time.Now()

	// Try to update existing record by peer_id (since peer_id is now unique)
	result, err := d.db.Exec(`
		UPDATE connections 
		SET address = ?, last_connected = ?, connection_type = ?, is_validated = ?, peer_name = CASE WHEN ? != '' THEN ? ELSE peer_name END
		WHERE peer_id = ?
	`, address, now, connectionType, isValidated, peerName, peerName, peerID)

	if err != nil {
		return fmt.Errorf("failed to update connection: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// If no rows were updated, insert new record
	if rowsAffected == 0 {
		_, err = d.db.Exec(`
			INSERT INTO connections (peer_id, address, first_connected, last_connected, connection_type, is_validated, peer_name)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, peerID, address, now, now, connectionType, isValidated, peerName)

		if err != nil {
			return fmt.Errorf("failed to insert connection: %w", err)
		}
		nameDisplay := peerName
		if nameDisplay == "" {
			nameDisplay = "unknown"
		}
		log.Printf("📝 New connection recorded: %s (%s) - %s", peerID[:12]+"...", connectionType, nameDisplay)
	} else {
		nameDisplay := peerName
		if nameDisplay == "" {
			nameDisplay = "unknown"
		}
		log.Printf("📝 Connection updated: %s (%s) - %s, new address: %s", peerID[:12]+"...", connectionType, nameDisplay, address)
	}

	return nil
}

// GetConnectionHistory retrieves all connection records
func (d *DatabaseService) GetConnectionHistory() ([]ConnectionRecord, error) {
	rows, err := d.db.Query(`
		SELECT id, peer_id, address, first_connected, last_connected, connection_type, is_validated, peer_name
		FROM connections
		ORDER BY last_connected DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query connections: %w", err)
	}
	defer rows.Close()

	var connections []ConnectionRecord
	for rows.Next() {
		var conn ConnectionRecord
		err := rows.Scan(
			&conn.ID, &conn.PeerID, &conn.Address,
			&conn.FirstConnected, &conn.LastConnected,
			&conn.ConnectionType, &conn.IsValidated, &conn.PeerName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan connection: %w", err)
		}
		connections = append(connections, conn)
	}

	return connections, nil
}

// GetRecentConnections retrieves connections from the last N days
func (d *DatabaseService) GetRecentConnections(days int) ([]ConnectionRecord, error) {
	cutoff := time.Now().AddDate(0, 0, -days)

	rows, err := d.db.Query(`
		SELECT id, peer_id, address, first_connected, last_connected, connection_type, is_validated, peer_name
		FROM connections
		WHERE last_connected >= ?
		ORDER BY last_connected DESC
	`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent connections: %w", err)
	}
	defer rows.Close()

	var connections []ConnectionRecord
	for rows.Next() {
		var conn ConnectionRecord
		err := rows.Scan(
			&conn.ID, &conn.PeerID, &conn.Address,
			&conn.FirstConnected, &conn.LastConnected,
			&conn.ConnectionType, &conn.IsValidated, &conn.PeerName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan connection: %w", err)
		}
		connections = append(connections, conn)
	}

	return connections, nil
}

// AddFriend adds a peer to the friends list
func (d *DatabaseService) AddFriend(peerID, peerName string) error {
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO friends (peer_id, peer_name, added_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`, peerID, peerName)
	if err != nil {
		return fmt.Errorf("failed to add friend: %w", err)
	}

	log.Printf("👥 Added friend: %s (%s)", peerName, peerID)
	return nil
}

// RemoveFriend removes a peer from the friends list
func (d *DatabaseService) RemoveFriend(peerID string) error {
	result, err := d.db.Exec("DELETE FROM friends WHERE peer_id = ?", peerID)
	if err != nil {
		return fmt.Errorf("failed to remove friend: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("friend not found")
	}

	log.Printf("👥 Removed friend: %s", peerID)
	return nil
}

// GetFriends retrieves all friends
func (d *DatabaseService) GetFriends() ([]models.Friend, error) {
	rows, err := d.db.Query(`
		SELECT id, peer_id, peer_name, added_at, last_seen, is_online
		FROM friends
		ORDER BY peer_name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query friends: %w", err)
	}
	defer rows.Close()

	var friends []models.Friend
	for rows.Next() {
		var friend models.Friend
		var lastSeen sql.NullTime

		err := rows.Scan(
			&friend.ID, &friend.PeerID, &friend.PeerName,
			&friend.AddedAt, &lastSeen, &friend.IsOnline,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan friend: %w", err)
		}

		if lastSeen.Valid {
			friend.LastSeen = &lastSeen.Time
		}

		friends = append(friends, friend)
	}

	return friends, nil
}

// IsFriend checks if a peer is in the friends list
func (d *DatabaseService) IsFriend(peerID string) (bool, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM friends WHERE peer_id = ?", peerID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if peer is friend: %w", err)
	}
	return count > 0, nil
}

// UpdateFriendStatus updates a friend's online status and last seen time
func (d *DatabaseService) UpdateFriendStatus(peerID string, isOnline bool) error {
	_, err := d.db.Exec(`
		UPDATE friends 
		SET is_online = ?, last_seen = CURRENT_TIMESTAMP
		WHERE peer_id = ?
	`, isOnline, peerID)
	if err != nil {
		return fmt.Errorf("failed to update friend status: %w", err)
	}
	return nil
}

// computeFileHash computes BLAKE3 hash of a file
func computeFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hasher := blake3.New(32, nil)
	_, err = io.Copy(hasher, file)
	if err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// getFileType determines if a file is a note or image based on extension
func getFileType(extension string) string {
	imageExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".bmp": true, ".tiff": true, ".webp": true,
	}

	if imageExts[strings.ToLower(extension)] {
		return "image"
	}
	return "note"
}

// ScanFiles scans the space184/notes and space184/images directories and updates the files table
func (d *DatabaseService) ScanFiles() error {
	log.Printf("🔍 Starting file scan...")

	// Get user home directory for proper path resolution
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Define directories to scan with full paths
	scanDirs := []string{
		filepath.Join(homeDir, "space184", "notes"),
		filepath.Join(homeDir, "space184", "images"),
	}

	for _, dir := range scanDirs {
		if err := d.scanDirectory(dir); err != nil {
			log.Printf("⚠️ Warning: failed to scan directory %s: %v", dir, err)
			// Continue scanning other directories even if one fails
		}
	}

	log.Printf("✅ File scan completed")
	return nil
}

// scanDirectory scans a specific directory for files
func (d *DatabaseService) scanDirectory(dirPath string) error {
	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		log.Printf("📁 Directory %s does not exist, skipping", dirPath)
		return nil
	}

	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("⚠️ Error accessing path %s: %v", path, err)
			return nil // Continue walking even if there's an error with one file
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get file extension
		extension := strings.ToLower(filepath.Ext(path))
		if extension == "" {
			return nil // Skip files without extensions
		}

		// Compute relative path from user home directory
		var relPath string
		homeDir, err := os.UserHomeDir()
		if err != nil {
			relPath = path // Fall back to absolute path if home dir fails
		} else {
			relPath, err = filepath.Rel(homeDir, path)
			if err != nil {
				relPath = path // Fall back to absolute path if relative fails
			}
		}

		// Check if file already exists in database
		exists, currentHash, err := d.fileExistsInDB(relPath)
		if err != nil {
			log.Printf("⚠️ Error checking file in database %s: %v", relPath, err)
			return nil
		}

		// Compute file hash
		hash, err := computeFileHash(path)
		if err != nil {
			log.Printf("⚠️ Error computing hash for %s: %v", relPath, err)
			return nil
		}

		// If file exists and hash hasn't changed, skip
		if exists && currentHash == hash {
			return nil
		}

		// Determine file type
		fileType := getFileType(extension)

		// Insert or update file record
		if err := d.upsertFileRecord(relPath, hash, info.Size(), extension, fileType); err != nil {
			log.Printf("⚠️ Error upserting file record for %s: %v", relPath, err)
			return nil
		}

		if exists {
			log.Printf("📝 Updated file: %s", relPath)
		} else {
			log.Printf("📄 Added file: %s (%s)", relPath, fileType)
		}

		return nil
	})
}

// fileExistsInDB checks if a file exists in the database and returns its current hash
func (d *DatabaseService) fileExistsInDB(filePath string) (bool, string, error) {

	var hash string
	err := d.db.QueryRow("SELECT hash FROM files WHERE filepath = ?", filePath).Scan(&hash)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, "", nil
		}
		return false, "", fmt.Errorf("failed to check file existence: %w", err)
	}

	return true, hash, nil
}

// upsertFileRecord inserts or updates a file record
func (d *DatabaseService) upsertFileRecord(filePath, hash string, size int64, extension, fileType string) error {
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO files (filepath, hash, size, extension, type, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, filePath, hash, size, extension, fileType)

	if err != nil {
		return fmt.Errorf("failed to upsert file record: %w", err)
	}
	return nil
}

// GetFiles retrieves all files from the database
func (d *DatabaseService) GetFiles() ([]FileRecord, error) {
	rows, err := d.db.Query(`
		SELECT id, filepath, hash, size, extension, type, updated_at
		FROM files
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query files: %w", err)
	}
	defer rows.Close()

	var files []FileRecord
	for rows.Next() {
		var file FileRecord
		err := rows.Scan(
			&file.ID, &file.FilePath, &file.Hash,
			&file.Size, &file.Extension, &file.Type, &file.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}
		files = append(files, file)
	}

	return files, nil
}

// CleanupDeletedFiles removes file records for files that no longer exist on disk
func (d *DatabaseService) CleanupDeletedFiles() error {
	files, err := d.GetFiles()
	if err != nil {
		return fmt.Errorf("failed to get files for cleanup: %w", err)
	}

	deletedCount := 0
	for _, file := range files {

		homeDir, _ := os.UserHomeDir()
		var relPath = filepath.Join(homeDir, file.FilePath)

		if _, err := os.Stat(relPath); os.IsNotExist(err) {
			// File no longer exists, remove from database
			_, err := d.db.Exec("DELETE FROM files WHERE id = ?", file.ID)
			if err != nil {
				log.Printf("⚠️ Failed to delete file record for %s: %v", file.FilePath, err)
				continue
			}
			log.Printf("🗑️ Removed deleted file: %s", file.FilePath)
			deletedCount++
		}
	}

	if deletedCount > 0 {
		log.Printf("🧹 Cleaned up %d deleted file records", deletedCount)
	}

	return nil
}

// Close closes the database connection
func (d *DatabaseService) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}
