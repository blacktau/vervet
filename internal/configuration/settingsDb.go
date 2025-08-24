package configuration

import (
	"database/sql"
	"fmt"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
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
		is_group BOOLEAN NOT NULL DEFAULT 0,
		UNIQUE(name, parent_id)
	)
	`
	_, err := d.db.Exec(query)
	return err
}

func (d *SettingsDatabase) GetRegisteredServersTree() ([]RegisteredServer, error) {
	rows, err := d.db.Query("SELECT id, name, parent_id, is_group FROM registered_servers ORDER BY is_group DESC, name ASC")
	if err != nil {
		return nil, fmt.Errorf("failed to query registered servers: %w", err)
	}

	defer rows.Close()

	var registeredServers []RegisteredServer
	for rows.Next() {
		var conn RegisteredServer
		if err := rows.Scan(&conn.ID, &conn.Name, &conn.ParentID, &conn.IsGroup); err != nil {
			return nil, fmt.Errorf("failed to read registered server: %w", err)
		}

		registeredServers = append(registeredServers, conn)
	}
	return registeredServers, nil
}

func (d *SettingsDatabase) CreateGroup(parentID int, name string) (int64, error) {
	result, err := d.db.Exec("INSERT INTO registered_servers (name, parent_id, is_group) VALUES (?, ?, 1)", name, parentID)
	if err != nil {
		return 0, fmt.Errorf("failed to create group: %w", err)
	}

	id, _ := result.LastInsertId()
	return id, nil
}

func (d *SettingsDatabase) UpdateGroup(groupID, parentID int, name string) error {
	_, err := d.db.Exec("UPDATE registered_servers SET name = ?, parent_id = ? WHERE id = ?", name, parentID, groupID)
	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	return nil
}

func (d *SettingsDatabase) SaveRegisteredServer(parentID int, name string) (int, error) {
	result, err := d.db.Exec("INSERT into registered_servers (name, parent_id, is_group) VALUES (?, ?, 0)", name, parentID)
	if err != nil {
		return 0, fmt.Errorf("failed to save registered server metadata: %w", err)
	}

	id, _ := result.LastInsertId()
	return int(id), nil
}

func (d *SettingsDatabase) UpdateRegisteredServer(serverID, parentID int, name string) error {
	_, err := d.db.Exec("UPDATE registered_servers SET name = ?, parent_id = ? WHERE id = ?", name, parentID, serverID)
	if err != nil {
		return fmt.Errorf("failed to update registered server: %w", err)
	}

	return nil
}

func (d *SettingsDatabase) DeleteNode(id int) error {
	var isGroup bool
	var count int

	err := d.db.QueryRow("SELECT is_group FROM registered_servers WHERE id = ?", id).Scan(&isGroup)
	if err != nil {
		return fmt.Errorf("node not found or query error: %w", err)
	}

	if isGroup {
		err := d.db.QueryRow("SELECT COUNT(*) FROM registered_servers WHERE parent_id = ?", id).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check group contents : %w", err)
		}
		if count > 0 {
			return fmt.Errorf("cannot delete a non-empty group. Please move or delete contents first")
		}
	}

	_, err = d.db.Exec("DELETE from registered_servers WHERE id = ?", id)

	return err
}
