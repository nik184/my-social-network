package repository

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	_ "modernc.org/sqlite"

	"my-social-network/internal/interfaces"
	"my-social-network/internal/models"
	"my-social-network/internal/utils"
)

// SQLiteRepository implements all repository interfaces using SQLite
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a new SQLite repository
func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
	// Ensure the directory exists
	dir := utils.DefaultPathManager.GetSpace184Path()
	if err := utils.EnsureDir(dir); err != nil {
		return nil, utils.WrapDatabaseError("create_directory", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, utils.WrapDatabaseError("open_database", err)
	}

	repo := &SQLiteRepository{db: db}

	if err := repo.initializeTables(); err != nil {
		db.Close()
		return nil, utils.WrapDatabaseError("initialize_tables", err)
	}

	if err := repo.initializeDefaultSettings(); err != nil {
		db.Close()
		return nil, utils.WrapDatabaseError("initialize_settings", err)
	}

	return repo, nil
}

// initializeTables creates all required tables
func (r *SQLiteRepository) initializeTables() error {
	tables := []struct {
		name string
		sql  string
	}{
		{"settings", r.getSettingsTableSQL()},
		{"connections", r.getConnectionsTableSQL()},
		{"friends", r.getFriendsTableSQL()},
		{"files", r.getFilesTableSQL()},
	}

	for _, table := range tables {
		if _, err := r.db.Exec(table.sql); err != nil {
			return fmt.Errorf("failed to create %s table: %w", table.name, err)
		}
	}

	log.Printf("📊 Database tables initialized successfully")
	return nil
}

func (r *SQLiteRepository) getSettingsTableSQL() string {
	return `CREATE TABLE IF NOT EXISTS settings (
		key VARCHAR(255) PRIMARY KEY,
		value TEXT NOT NULL
	);`
}

func (r *SQLiteRepository) getConnectionsTableSQL() string {
	return `CREATE TABLE IF NOT EXISTS connections (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		peer_id VARCHAR(255) NOT NULL UNIQUE,
		address VARCHAR(255) NOT NULL,
		first_connected DATETIME NOT NULL,
		last_connected DATETIME NOT NULL,
		connection_type VARCHAR(255) NOT NULL,
		is_validated BOOLEAN NOT NULL DEFAULT 0,
		peer_name VARCHAR(255) DEFAULT ''
	);`
}

func (r *SQLiteRepository) getFriendsTableSQL() string {
	return `CREATE TABLE IF NOT EXISTS friends (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		peer_id VARCHAR(255) NOT NULL UNIQUE,
		peer_name VARCHAR(255) NOT NULL,
		added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_seen DATETIME,
		is_online BOOLEAN NOT NULL DEFAULT 0
	);`
}

func (r *SQLiteRepository) getFilesTableSQL() string {
	return `CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		filepath VARCHAR(255) NOT NULL,
		hash VARCHAR(255) NOT NULL,
		size INTEGER NOT NULL,
		extension VARCHAR(255) NOT NULL,
		type VARCHAR(255) NOT NULL,
		peer_id VARCHAR(255) NOT NULL,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(filepath, peer_id)
	);`
}

// initializeDefaultSettings creates default settings if they don't exist
func (r *SQLiteRepository) initializeDefaultSettings() error {
	// Check if settings already exist
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM settings").Scan(&count)
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
		"private_key": string(privKeyBytes),
	}

	for key, value := range settings {
		if err := r.SetSetting(key, value); err != nil {
			return fmt.Errorf("failed to insert setting %s: %w", key, err)
		}
	}

	log.Printf("🎯 Created default settings: name=jerry, node_id=%s", nodeID.String())
	return nil
}

// Settings Repository Implementation
func (r *SQLiteRepository) GetSetting(key string) (string, error) {
	var value string
	err := r.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", utils.NewNotFoundError("setting", key)
		}
		return "", utils.WrapDatabaseError("get_setting", err)
	}
	return value, nil
}

func (r *SQLiteRepository) SetSetting(key, value string) error {
	_, err := r.db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value)
	if err != nil {
		return utils.WrapDatabaseError("set_setting", err)
	}
	return nil
}

func (r *SQLiteRepository) GetAllSettings() (map[string]string, error) {
	rows, err := r.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, utils.WrapDatabaseError("get_all_settings", err)
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, utils.WrapDatabaseError("scan_setting", err)
		}
		settings[key] = value
	}

	return settings, nil
}

// Connection Repository Implementation
func (r *SQLiteRepository) RecordConnection(peerID, address, connectionType string, isValidated bool) error {
	return r.RecordConnectionWithName(peerID, address, connectionType, isValidated, "")
}

func (r *SQLiteRepository) RecordConnectionWithName(peerID, address, connectionType string, isValidated bool, peerName string) error {
	now := time.Now()

	// Try to update existing record by peer_id (since peer_id is now unique)
	result, err := r.db.Exec(`
		UPDATE connections 
		SET address = ?, last_connected = ?, connection_type = ?, is_validated = ?, peer_name = CASE WHEN ? != '' THEN ? ELSE peer_name END
		WHERE peer_id = ?
	`, address, now, connectionType, isValidated, peerName, peerName, peerID)

	if err != nil {
		return utils.WrapDatabaseError("update_connection", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return utils.WrapDatabaseError("get_rows_affected", err)
	}

	// If no rows were updated, insert new record
	if rowsAffected == 0 {
		_, err = r.db.Exec(`
			INSERT INTO connections (peer_id, address, first_connected, last_connected, connection_type, is_validated, peer_name)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, peerID, address, now, now, connectionType, isValidated, peerName)

		if err != nil {
			return utils.WrapDatabaseError("insert_connection", err)
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

func (r *SQLiteRepository) GetConnectionHistory() ([]models.ConnectionRecord, error) {
	rows, err := r.db.Query(`
		SELECT id, peer_id, address, first_connected, last_connected, connection_type, is_validated, peer_name
		FROM connections
		ORDER BY last_connected DESC
	`)
	if err != nil {
		return nil, utils.WrapDatabaseError("get_connection_history", err)
	}
	defer rows.Close()

	var connections []models.ConnectionRecord
	for rows.Next() {
		var conn models.ConnectionRecord
		err := rows.Scan(
			&conn.ID, &conn.PeerID, &conn.Address,
			&conn.FirstConnected, &conn.LastConnected,
			&conn.ConnectionType, &conn.IsValidated, &conn.PeerName,
		)
		if err != nil {
			return nil, utils.WrapDatabaseError("scan_connection", err)
		}
		connections = append(connections, conn)
	}

	return connections, nil
}

func (r *SQLiteRepository) GetRecentConnections(days int) ([]models.ConnectionRecord, error) {
	cutoff := time.Now().AddDate(0, 0, -days)

	rows, err := r.db.Query(`
		SELECT id, peer_id, address, first_connected, last_connected, connection_type, is_validated, peer_name
		FROM connections
		WHERE last_connected >= ?
		ORDER BY last_connected DESC
	`, cutoff)
	if err != nil {
		return nil, utils.WrapDatabaseError("get_recent_connections", err)
	}
	defer rows.Close()

	var connections []models.ConnectionRecord
	for rows.Next() {
		var conn models.ConnectionRecord
		err := rows.Scan(
			&conn.ID, &conn.PeerID, &conn.Address,
			&conn.FirstConnected, &conn.LastConnected,
			&conn.ConnectionType, &conn.IsValidated, &conn.PeerName,
		)
		if err != nil {
			return nil, utils.WrapDatabaseError("scan_connection", err)
		}
		connections = append(connections, conn)
	}

	return connections, nil
}

// Friends Repository Implementation
func (r *SQLiteRepository) AddFriend(peerID, peerName string) error {
	_, err := r.db.Exec(`
		INSERT OR REPLACE INTO friends (peer_id, peer_name, added_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`, peerID, peerName)
	if err != nil {
		return utils.WrapDatabaseError("add_friend", err)
	}

	log.Printf("👥 Added friend: %s (%s)", peerName, peerID)
	return nil
}

func (r *SQLiteRepository) RemoveFriend(peerID string) error {
	result, err := r.db.Exec("DELETE FROM friends WHERE peer_id = ?", peerID)
	if err != nil {
		return utils.WrapDatabaseError("remove_friend", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return utils.WrapDatabaseError("get_rows_affected", err)
	}

	if rowsAffected == 0 {
		return utils.NewNotFoundError("friend", peerID)
	}

	log.Printf("👥 Removed friend: %s", peerID)
	return nil
}

func (r *SQLiteRepository) GetFriends() ([]models.Friend, error) {
	rows, err := r.db.Query(`
		SELECT id, peer_id, peer_name, added_at, last_seen, is_online
		FROM friends
		ORDER BY peer_name ASC
	`)
	if err != nil {
		return nil, utils.WrapDatabaseError("get_friends", err)
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
			return nil, utils.WrapDatabaseError("scan_friend", err)
		}

		if lastSeen.Valid {
			friend.LastSeen = &lastSeen.Time
		}

		friends = append(friends, friend)
	}

	return friends, nil
}

func (r *SQLiteRepository) IsFriend(peerID string) (bool, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM friends WHERE peer_id = ?", peerID).Scan(&count)
	if err != nil {
		return false, utils.WrapDatabaseError("check_friend", err)
	}
	return count > 0, nil
}

func (r *SQLiteRepository) UpdateFriendStatus(peerID string, isOnline bool) error {
	_, err := r.db.Exec(`
		UPDATE friends 
		SET is_online = ?, last_seen = CURRENT_TIMESTAMP
		WHERE peer_id = ?
	`, isOnline, peerID)
	if err != nil {
		return utils.WrapDatabaseError("update_friend_status", err)
	}
	return nil
}

// Files Repository Implementation
func (r *SQLiteRepository) FileExists(filePath string) (bool, string, error) {
	var hash string
	err := r.db.QueryRow("SELECT hash FROM files WHERE filepath = ?", filePath).Scan(&hash)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, "", nil
		}
		return false, "", utils.WrapDatabaseError("check_file_exists", err)
	}
	return true, hash, nil
}

func (r *SQLiteRepository) UpsertFileRecord(filePath, hash string, size int64, extension, fileType, peerID string) error {
	_, err := r.db.Exec(`
		INSERT OR REPLACE INTO files (filepath, hash, size, extension, type, peer_id, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, filePath, hash, size, extension, fileType, peerID)

	if err != nil {
		return utils.WrapDatabaseError("upsert_file_record", err)
	}
	return nil
}

func (r *SQLiteRepository) GetFiles() ([]models.FileRecord, error) {
	rows, err := r.db.Query(`
		SELECT id, filepath, hash, size, extension, type, peer_id, updated_at
		FROM files
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, utils.WrapDatabaseError("get_files", err)
	}
	defer rows.Close()

	var files []models.FileRecord
	for rows.Next() {
		var file models.FileRecord
		err := rows.Scan(
			&file.ID, &file.FilePath, &file.Hash,
			&file.Size, &file.Extension, &file.Type, &file.PeerID, &file.UpdatedAt,
		)
		if err != nil {
			return nil, utils.WrapDatabaseError("scan_file", err)
		}
		files = append(files, file)
	}

	return files, nil
}

func (r *SQLiteRepository) DeleteFileRecord(fileID int) error {
	_, err := r.db.Exec("DELETE FROM files WHERE id = ?", fileID)
	if err != nil {
		return utils.WrapDatabaseError("delete_file_record", err)
	}
	return nil
}

// Additional methods needed by the current system
func (r *SQLiteRepository) GetNodePrivateKey() (crypto.PrivKey, error) {
	privKeyStr, err := r.GetSetting("private_key")
	if err != nil {
		return nil, fmt.Errorf("failed to get private key setting: %w", err)
	}

	privKey, err := crypto.UnmarshalPrivateKey([]byte(privKeyStr))
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal private key: %w", err)
	}

	return privKey, nil
}

func (r *SQLiteRepository) GetNodeID() (peer.ID, error) {
	nodeIDStr, err := r.GetSetting("node_id")
	if err != nil {
		return "", fmt.Errorf("failed to get node ID setting: %w", err)
	}

	nodeID, err := peer.Decode(nodeIDStr)
	if err != nil {
		return "", fmt.Errorf("failed to decode node ID: %w", err)
	}

	return nodeID, nil
}

// Close closes the database connection
func (r *SQLiteRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// Ensure SQLiteRepository implements all interfaces
var _ interfaces.DatabaseService = (*SQLiteRepository)(nil)
