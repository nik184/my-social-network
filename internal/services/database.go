package services

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	_ "github.com/mattn/go-sqlite3"
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

	db, err := sql.Open("sqlite3", dbPath)
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

	// Create connections table
	connectionsTable := `
		CREATE TABLE IF NOT EXISTS connections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			peer_id TEXT NOT NULL,
			address TEXT NOT NULL,
			first_connected DATETIME NOT NULL,
			last_connected DATETIME NOT NULL,
			connection_type TEXT NOT NULL,
			is_validated BOOLEAN NOT NULL DEFAULT 0,
			peer_name TEXT DEFAULT '',
			UNIQUE(peer_id, address)
		);
	`
	if _, err := d.db.Exec(connectionsTable); err != nil {
		return fmt.Errorf("failed to create connections table: %w", err)
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
	
	// Try to update existing record
	result, err := d.db.Exec(`
		UPDATE connections 
		SET last_connected = ?, connection_type = ?, is_validated = ?, peer_name = CASE WHEN ? != '' THEN ? ELSE peer_name END
		WHERE peer_id = ? AND address = ?
	`, now, connectionType, isValidated, peerName, peerName, peerID, address)
	
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
		log.Printf("📝 Connection updated: %s (%s) - %s", peerID[:12]+"...", connectionType, nameDisplay)
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

// Close closes the database connection
func (d *DatabaseService) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}