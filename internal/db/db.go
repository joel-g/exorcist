package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

// New creates a new database connection and runs migrations
func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	db := &DB{conn: conn}

	// Run migrations
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

func (db *DB) migrate() error {
	_, err := db.conn.Exec(Schema)
	return err
}

func (db *DB) Close() error {
	return db.conn.Close()
}

// UpsertGroup creates or updates a group
func (db *DB) UpsertGroup(signalGroupID, name string, inactivityDays int) (int64, error) {
	result, err := db.conn.Exec(`
		INSERT INTO groups (signal_group_id, name, inactivity_days)
		VALUES (?, ?, ?)
		ON CONFLICT(signal_group_id) DO UPDATE SET
			name = excluded.name,
			inactivity_days = excluded.inactivity_days
	`, signalGroupID, name, inactivityDays)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// UpsertGroupMember creates or updates a group member
func (db *DB) UpsertGroupMember(groupID int64, signalUserID string) error {
	_, err := db.conn.Exec(`
		INSERT INTO group_members (group_id, signal_user_id)
		VALUES (?, ?)
		ON CONFLICT(group_id, signal_user_id) DO NOTHING
	`, groupID, signalUserID)
	return err
}

// UpsertGroupMemberWithTimestamp creates or updates a group member and sets last_message_at
func (db *DB) UpsertGroupMemberWithTimestamp(groupID int64, signalUserID string, timestamp time.Time) error {
	_, err := db.conn.Exec(`
		INSERT INTO group_members (group_id, signal_user_id, last_message_at)
		VALUES (?, ?, ?)
		ON CONFLICT(group_id, signal_user_id) DO UPDATE SET
			last_message_at = excluded.last_message_at
	`, groupID, signalUserID, timestamp)
	return err
}

// UpdateLastMessage updates the last_message_at timestamp for a member
func (db *DB) UpdateLastMessage(groupID int64, signalUserID string, timestamp time.Time) error {
	_, err := db.conn.Exec(`
		UPDATE group_members
		SET last_message_at = ?
		WHERE group_id = ? AND signal_user_id = ?
	`, timestamp, groupID, signalUserID)
	return err
}

// GetInactiveMembers returns members who haven't messaged in N days
func (db *DB) GetInactiveMembers(groupID int64, inactivityDays int) ([]GroupMember, error) {
	cutoff := time.Now().AddDate(0, 0, -inactivityDays)

	rows, err := db.conn.Query(`
		SELECT id, group_id, signal_user_id, last_message_at, joined_at, created_at
		FROM group_members
		WHERE group_id = ?
		  AND (last_message_at IS NULL OR last_message_at < ?)
		  AND joined_at < ?
	`, groupID, cutoff, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []GroupMember
	for rows.Next() {
		var m GroupMember
		err := rows.Scan(&m.ID, &m.GroupID, &m.SignalUserID,
			&m.LastMessageAt, &m.JoinedAt, &m.CreatedAt)
		if err != nil {
			return nil, err
		}
		members = append(members, m)
	}

	return members, rows.Err()
}

// GetAllGroups returns all monitored groups
func (db *DB) GetAllGroups() ([]Group, error) {
	rows, err := db.conn.Query(`
		SELECT id, signal_group_id, name, inactivity_days, created_at
		FROM groups
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []Group
	for rows.Next() {
		var g Group
		err := rows.Scan(&g.ID, &g.SignalGroupID, &g.Name, &g.InactivityDays, &g.CreatedAt)
		if err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}

	return groups, rows.Err()
}

// RemoveMember deletes a member from the database
func (db *DB) RemoveMember(groupID int64, signalUserID string) error {
	_, err := db.conn.Exec(`
		DELETE FROM group_members
		WHERE group_id = ? AND signal_user_id = ?
	`, groupID, signalUserID)
	return err
}

// CountGroupMembers returns the number of members in a group
func (db *DB) CountGroupMembers(groupID int64) (int, error) {
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM group_members WHERE group_id = ?
	`, groupID).Scan(&count)
	return count, err
}

// SetMemberJoinedAt updates the joined_at timestamp for a member (for testing)
func (db *DB) SetMemberJoinedAt(groupID int64, signalUserID string, joinedAt time.Time) error {
	_, err := db.conn.Exec(`
		UPDATE group_members
		SET joined_at = ?
		WHERE group_id = ? AND signal_user_id = ?
	`, joinedAt, groupID, signalUserID)
	return err
}
