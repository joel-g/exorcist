package db

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// setupTestDB creates a temporary test database
func setupTestDB(t *testing.T) *DB {
	t.Helper()

	// Create temporary database file
	tmpFile, err := os.CreateTemp("", "exorcist-test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp db: %v", err)
	}
	tmpFile.Close()

	// Initialize database
	db, err := New(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("Failed to initialize db: %v", err)
	}

	// Clean up when test completes
	t.Cleanup(func() {
		db.Close()
		os.Remove(tmpFile.Name())
	})

	return db
}

func TestNew(t *testing.T) {
	db := setupTestDB(t)

	if db == nil {
		t.Fatal("Expected non-nil database")
	}

	// Verify tables were created
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='table' AND name IN ('groups', 'group_members')
	`).Scan(&count)

	if err != nil {
		t.Fatalf("Failed to query tables: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 tables, got %d", count)
	}
}

func TestUpsertGroup(t *testing.T) {
	db := setupTestDB(t)

	tests := []struct {
		name           string
		signalGroupID  string
		groupName      string
		inactivityDays int
		wantErr        bool
	}{
		{
			name:           "create new group",
			signalGroupID:  "test-group-123",
			groupName:      "Test Group",
			inactivityDays: 30,
			wantErr:        false,
		},
		{
			name:           "update existing group",
			signalGroupID:  "test-group-123",
			groupName:      "Updated Test Group",
			inactivityDays: 45,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := db.UpsertGroup(tt.signalGroupID, tt.groupName, tt.inactivityDays)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpsertGroup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && id == 0 {
				t.Error("Expected non-zero ID")
			}

			// Verify the group was created/updated
			var name string
			var days int
			err = db.conn.QueryRow(`
				SELECT name, inactivity_days
				FROM groups
				WHERE signal_group_id = ?
			`, tt.signalGroupID).Scan(&name, &days)

			if err != nil {
				t.Fatalf("Failed to query group: %v", err)
			}

			if name != tt.groupName {
				t.Errorf("Expected name %q, got %q", tt.groupName, name)
			}

			if days != tt.inactivityDays {
				t.Errorf("Expected inactivity days %d, got %d", tt.inactivityDays, days)
			}
		})
	}
}

func TestUpsertGroupMember(t *testing.T) {
	db := setupTestDB(t)

	// Create a test group first
	groupID, err := db.UpsertGroup("test-group-1", "Test Group", 30)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	tests := []struct {
		name         string
		groupID      int64
		signalUserID string
		wantErr      bool
	}{
		{
			name:         "add new member",
			groupID:      groupID,
			signalUserID: "user-uuid-1",
			wantErr:      false,
		},
		{
			name:         "add duplicate member (should be ignored)",
			groupID:      groupID,
			signalUserID: "user-uuid-1",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.UpsertGroupMember(tt.groupID, tt.signalUserID)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpsertGroupMember() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Verify only one member exists (duplicate should be ignored)
	count, err := db.CountGroupMembers(groupID)
	if err != nil {
		t.Fatalf("Failed to count members: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 member, got %d", count)
	}
}

func TestUpsertGroupMemberWithTimestamp(t *testing.T) {
	db := setupTestDB(t)

	groupID, err := db.UpsertGroup("test-group-2", "Test Group", 30)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	now := time.Now()
	userID := "user-uuid-2"

	// Insert with timestamp
	err = db.UpsertGroupMemberWithTimestamp(groupID, userID, now)
	if err != nil {
		t.Fatalf("Failed to upsert member: %v", err)
	}

	// Verify timestamp was set
	var lastMessageAt time.Time
	err = db.conn.QueryRow(`
		SELECT last_message_at
		FROM group_members
		WHERE group_id = ? AND signal_user_id = ?
	`, groupID, userID).Scan(&lastMessageAt)

	if err != nil {
		t.Fatalf("Failed to query member: %v", err)
	}

	// Allow 1 second difference for test timing
	if lastMessageAt.Sub(now).Abs() > time.Second {
		t.Errorf("Expected timestamp %v, got %v", now, lastMessageAt)
	}

	// Update with newer timestamp
	later := now.Add(1 * time.Hour)
	err = db.UpsertGroupMemberWithTimestamp(groupID, userID, later)
	if err != nil {
		t.Fatalf("Failed to update member: %v", err)
	}

	// Verify timestamp was updated
	err = db.conn.QueryRow(`
		SELECT last_message_at
		FROM group_members
		WHERE group_id = ? AND signal_user_id = ?
	`, groupID, userID).Scan(&lastMessageAt)

	if err != nil {
		t.Fatalf("Failed to query updated member: %v", err)
	}

	if lastMessageAt.Sub(later).Abs() > time.Second {
		t.Errorf("Expected updated timestamp %v, got %v", later, lastMessageAt)
	}
}

func TestGetInactiveMembers(t *testing.T) {
	db := setupTestDB(t)

	groupID, err := db.UpsertGroup("test-group-3", "Test Group", 30)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	now := time.Now()

	// Add members with different activity levels
	// Active member - messaged recently
	err = db.UpsertGroupMemberWithTimestamp(groupID, "active-user", now)
	if err != nil {
		t.Fatalf("Failed to add active user: %v", err)
	}

	// Inactive member - messaged 40 days ago, joined 40 days ago
	oldTime := now.AddDate(0, 0, -40)
	err = db.UpsertGroupMemberWithTimestamp(groupID, "inactive-user", oldTime)
	if err != nil {
		t.Fatalf("Failed to add inactive user: %v", err)
	}

	// Update inactive user's joined_at to be old
	_, err = db.conn.Exec(`
		UPDATE group_members
		SET joined_at = ?
		WHERE signal_user_id = ?
	`, oldTime, "inactive-user")
	if err != nil {
		t.Fatalf("Failed to update inactive user joined_at: %v", err)
	}

	// Never messaged member - added 40 days ago
	err = db.UpsertGroupMember(groupID, "ghost-user")
	if err != nil {
		t.Fatalf("Failed to add ghost user: %v", err)
	}

	// Update ghost user's joined_at to be old
	_, err = db.conn.Exec(`
		UPDATE group_members
		SET joined_at = ?
		WHERE signal_user_id = ?
	`, oldTime, "ghost-user")
	if err != nil {
		t.Fatalf("Failed to update ghost user: %v", err)
	}

	// Get inactive members (30 day threshold)
	inactive, err := db.GetInactiveMembers(groupID, 30)
	if err != nil {
		t.Fatalf("Failed to get inactive members: %v", err)
	}

	// Should return 2 inactive members (inactive-user and ghost-user)
	if len(inactive) != 2 {
		t.Errorf("Expected 2 inactive members, got %d", len(inactive))
	}

	// Verify the inactive members are the right ones
	inactiveIDs := make(map[string]bool)
	for _, member := range inactive {
		inactiveIDs[member.SignalUserID] = true
	}

	if !inactiveIDs["inactive-user"] {
		t.Error("Expected inactive-user to be in inactive list")
	}

	if !inactiveIDs["ghost-user"] {
		t.Error("Expected ghost-user to be in inactive list")
	}

	if inactiveIDs["active-user"] {
		t.Error("Expected active-user NOT to be in inactive list")
	}
}

func TestGetAllGroups(t *testing.T) {
	db := setupTestDB(t)

	// Add multiple groups
	_, err := db.UpsertGroup("group-1", "Group One", 30)
	if err != nil {
		t.Fatalf("Failed to create group 1: %v", err)
	}

	_, err = db.UpsertGroup("group-2", "Group Two", 45)
	if err != nil {
		t.Fatalf("Failed to create group 2: %v", err)
	}

	// Get all groups
	groups, err := db.GetAllGroups()
	if err != nil {
		t.Fatalf("Failed to get groups: %v", err)
	}

	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}

	// Verify group data
	groupMap := make(map[string]Group)
	for _, g := range groups {
		groupMap[g.SignalGroupID] = g
	}

	g1, ok := groupMap["group-1"]
	if !ok {
		t.Error("Expected group-1 to exist")
	} else {
		if g1.Name != "Group One" {
			t.Errorf("Expected name 'Group One', got %q", g1.Name)
		}
		if g1.InactivityDays != 30 {
			t.Errorf("Expected inactivity days 30, got %d", g1.InactivityDays)
		}
	}

	g2, ok := groupMap["group-2"]
	if !ok {
		t.Error("Expected group-2 to exist")
	} else {
		if g2.Name != "Group Two" {
			t.Errorf("Expected name 'Group Two', got %q", g2.Name)
		}
		if g2.InactivityDays != 45 {
			t.Errorf("Expected inactivity days 45, got %d", g2.InactivityDays)
		}
	}
}

func TestRemoveMember(t *testing.T) {
	db := setupTestDB(t)

	groupID, err := db.UpsertGroup("test-group-4", "Test Group", 30)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	userID := "user-to-remove"
	err = db.UpsertGroupMember(groupID, userID)
	if err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}

	// Verify member exists
	count, err := db.CountGroupMembers(groupID)
	if err != nil {
		t.Fatalf("Failed to count members: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 member before removal, got %d", count)
	}

	// Remove member
	err = db.RemoveMember(groupID, userID)
	if err != nil {
		t.Fatalf("Failed to remove member: %v", err)
	}

	// Verify member was removed
	count, err = db.CountGroupMembers(groupID)
	if err != nil {
		t.Fatalf("Failed to count members after removal: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 members after removal, got %d", count)
	}
}

func TestCountGroupMembers(t *testing.T) {
	db := setupTestDB(t)

	groupID, err := db.UpsertGroup("test-group-5", "Test Group", 30)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Start with 0 members
	count, err := db.CountGroupMembers(groupID)
	if err != nil {
		t.Fatalf("Failed to count members: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 members initially, got %d", count)
	}

	// Add members
	for i := 1; i <= 5; i++ {
		err = db.UpsertGroupMember(groupID, fmt.Sprintf("user-%d", i))
		if err != nil {
			t.Fatalf("Failed to add member %d: %v", i, err)
		}
	}

	// Count should be 5
	count, err = db.CountGroupMembers(groupID)
	if err != nil {
		t.Fatalf("Failed to count members: %v", err)
	}

	if count != 5 {
		t.Errorf("Expected 5 members, got %d", count)
	}
}

func TestForeignKeyConstraint(t *testing.T) {
	db := setupTestDB(t)

	groupID, err := db.UpsertGroup("test-group-6", "Test Group", 30)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Add a member
	err = db.UpsertGroupMember(groupID, "user-1")
	if err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}

	// Delete the group
	_, err = db.conn.Exec("DELETE FROM groups WHERE id = ?", groupID)
	if err != nil {
		t.Fatalf("Failed to delete group: %v", err)
	}

	// Member should be automatically deleted due to CASCADE
	count, err := db.CountGroupMembers(groupID)
	if err != nil {
		t.Fatalf("Failed to count members: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected members to be cascade deleted, but got %d members", count)
	}
}
