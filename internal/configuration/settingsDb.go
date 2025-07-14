package configuration

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"path/filepath"
)

type SettingsDatabase struct {
	db *sql.DB
}

func NewSettingsDatabase() (*SettingsDatabase, error) {
	dbPath := filepath.Join(".", "settings.db")

	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open settings database: %w", err)
	}

	db := &SettingsDatabase{db: database}
	if err := db.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return db, nil
}

func (d *SettingsDatabase) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS registered_servers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		parent_id INTEGER NOT NULL DEFAULT 0,
		is_folder BOOLEAN NOT NULL DEFAULT 0,
		UNIQUE(name, parent_id)
	)
	`
	_, err := d.db.Exec(query)
	return err
}

func (d *SettingsDatabase) GetRegisteredServersTree() ([]RegisteredServer, error) {
	rows, err := d.db.Query("SELECT id, name, parent_id, is_folder FROM registered_servers ORDER BY is_folder DESC, name ASC")
	if err != nil {
		return nil, fmt.Errorf("failed to query connections: %w", err)
	}

	defer rows.Close()

	var connections []RegisteredServer
	for rows.Next() {
		var conn RegisteredServer
		if err := rows.Scan(&conn.ID, &conn.Name, &conn.ParentID, &conn.IsFolder); err != nil {
			return nil, fmt.Errorf("failed to read connection: %w", err)
		}

		connections = append(connections, conn)
	}
	return connections, nil
}

func (d *SettingsDatabase) CreateFolder(name string, parentID int) (int64, error) {
	result, err := d.db.Exec("INSERT INTO registered_servers (name, parent_id, is_folder) VALUES (?, ?, 1)", name, parentID)
	if err != nil {
		return 0, fmt.Errorf("failed to create folder: %w", err)
	}

	id, _ := result.LastInsertId()
	return id, nil
}

func (d *SettingsDatabase) SaveRegisteredServer(name string, parentID int) (int64, error) {
	result, err := d.db.Exec("INSERT into registered_servers (name, parent_id, is_folder) VALUES (?, ?, 0)", name, parentID)
	if err != nil {
		return 0, fmt.Errorf("failed to save connection metadata: %w", err)
	}

	id, _ := result.LastInsertId()
	return id, nil
}

func (d *SettingsDatabase) DeleteNode(id int) error {
	var isFolder, count int

	err := d.db.QueryRow("SELECT is_folder FROM registered_servers WHERE id = ?", id).Scan(&isFolder)
	if err != nil {
		return fmt.Errorf("node not found or query error: %w", err)
	}

	if isFolder == 1 {
		err := d.db.QueryRow("SELECT COUNT(*) FROM registered_servers WHERE parent_id = ?", id).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check folder contents : %w", err)
		}
		if count > 0 {
			return fmt.Errorf("cannot delete a non-empty folder. Please move or delete contents first")
		}
	}

	_, err = d.db.Exec("DELETE from registered_servers WHERE id = ?", id)

	return err
}
