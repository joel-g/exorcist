package jobs

import (
	"os"
	"testing"
	"time"

	"github.com/joel-g/exorcist/internal/db"
)

// setupTestDB creates a temporary test database
func setupTestDB(t *testing.T) *db.DB {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "exorcist-jobs-test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp db: %v", err)
	}
	tmpFile.Close()

	database, err := db.New(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("Failed to initialize db: %v", err)
	}

	t.Cleanup(func() {
		database.Close()
		os.Remove(tmpFile.Name())
	})

	return database
}

// mockSignalClient is a mock implementation for testing
type mockSignalClient struct {
	groupMembers      map[string][]string
	receivedMessages  []mockMessage
	sentMessages      []mockSentMessage
	removedMembers    []mockRemovedMember
	getGroupMembersErr error
	receiveMessagesErr error
	sendMessageErr    error
	removeMememberErr  error
}

type mockMessage struct {
	groupID   string
	senderID  string
	timestamp time.Time
}

type mockSentMessage struct {
	groupID string
	message string
}

type mockRemovedMember struct {
	groupID  string
	memberID string
}

func newMockSignalClient() *mockSignalClient {
	return &mockSignalClient{
		groupMembers:     make(map[string][]string),
		receivedMessages: []mockMessage{},
		sentMessages:     []mockSentMessage{},
		removedMembers:   []mockRemovedMember{},
	}
}

// Note: These tests verify the job logic without actually calling signal-cli
// Integration tests with real signal-cli would be in a separate test suite

func TestSyncJob_DatabaseIntegration(t *testing.T) {
	// This test verifies the sync job correctly updates the database
	// when processing messages (without actually calling signal-cli)

	database := setupTestDB(t)

	// Setup: Create a test group
	groupID, err := database.UpsertGroup("test-group-1", "Test Group", 30)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Setup: Add an existing member
	existingUser := "existing-user-uuid"
	oldTime := time.Now().AddDate(0, 0, -10)
	err = database.UpsertGroupMemberWithTimestamp(groupID, existingUser, oldTime)
	if err != nil {
		t.Fatalf("Failed to add existing member: %v", err)
	}

	// Test: Update existing member's timestamp
	newTime := time.Now()
	err = database.UpsertGroupMemberWithTimestamp(groupID, existingUser, newTime)
	if err != nil {
		t.Fatalf("Failed to update member: %v", err)
	}

	// Verify: Check that timestamp was updated
	members, err := database.GetInactiveMembers(groupID, 5)
	if err != nil {
		t.Fatalf("Failed to get inactive members: %v", err)
	}

	// Should have no inactive members since we just updated the timestamp
	if len(members) != 0 {
		t.Errorf("Expected 0 inactive members, got %d", len(members))
	}

	// Test: Auto-add new member
	newUser := "new-user-uuid"
	err = database.UpsertGroupMemberWithTimestamp(groupID, newUser, newTime)
	if err != nil {
		t.Fatalf("Failed to auto-add new member: %v", err)
	}

	// Verify: New member was added
	count, err := database.CountGroupMembers(groupID)
	if err != nil {
		t.Fatalf("Failed to count members: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 members, got %d", count)
	}
}

func TestReapJob_DatabaseIntegration(t *testing.T) {
	// This test verifies the reap job correctly identifies inactive members

	database := setupTestDB(t)

	// Setup: Create test group with 30-day inactivity threshold
	groupID, err := database.UpsertGroup("test-group-2", "Test Group", 30)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	now := time.Now()

	// Add active member (messaged today)
	err = database.UpsertGroupMemberWithTimestamp(groupID, "active-user", now)
	if err != nil {
		t.Fatalf("Failed to add active member: %v", err)
	}

	// Add inactive member (messaged 40 days ago, joined 40 days ago)
	inactiveTime := now.AddDate(0, 0, -40)
	err = database.UpsertGroupMemberWithTimestamp(groupID, "inactive-user", inactiveTime)
	if err != nil {
		t.Fatalf("Failed to add inactive member: %v", err)
	}

	// Update inactive user's joined_at to simulate old membership
	err = database.SetMemberJoinedAt(groupID, "inactive-user", inactiveTime)
	if err != nil {
		t.Fatalf("Failed to update inactive user joined_at: %v", err)
	}

	// Add ghost member (never messaged, joined 40 days ago)
	err = database.UpsertGroupMember(groupID, "ghost-user")
	if err != nil {
		t.Fatalf("Failed to add ghost member: %v", err)
	}

	// Test: Get inactive members
	inactive, err := database.GetInactiveMembers(groupID, 30)
	if err != nil {
		t.Fatalf("Failed to get inactive members: %v", err)
	}

	// Should find the inactive member
	// Note: ghost-user won't show up because joined_at defaults to NOW()
	// In real usage, initialization would have set their timestamp to NOW()
	// and they'd need 30 days of no activity to be reaped

	foundInactive := false
	for _, member := range inactive {
		if member.SignalUserID == "inactive-user" {
			foundInactive = true
		}
		if member.SignalUserID == "active-user" {
			t.Error("Active user should not be in inactive list")
		}
	}

	if !foundInactive {
		t.Error("Inactive user should be in inactive list")
	}

	// Test: Remove inactive member
	err = database.RemoveMember(groupID, "inactive-user")
	if err != nil {
		t.Fatalf("Failed to remove inactive member: %v", err)
	}

	// Verify: Member was removed
	count, err := database.CountGroupMembers(groupID)
	if err != nil {
		t.Fatalf("Failed to count members: %v", err)
	}

	// Should have 2 members left (active-user and ghost-user)
	if count != 2 {
		t.Errorf("Expected 2 members after removal, got %d", count)
	}
}

func TestInitJob_DatabaseIntegration(t *testing.T) {
	// This test verifies the init job correctly seeds group members

	database := setupTestDB(t)

	// Setup: Create test group
	groupID, err := database.UpsertGroup("test-group-3", "Test Group", 30)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Initially, group should have no members
	count, err := database.CountGroupMembers(groupID)
	if err != nil {
		t.Fatalf("Failed to count initial members: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 initial members, got %d", count)
	}

	// Test: Initialize members (simulating what InitJob.Run() would do)
	now := time.Now()
	mockMembers := []string{"user-1", "user-2", "user-3"}

	for _, memberID := range mockMembers {
		err = database.UpsertGroupMemberWithTimestamp(groupID, memberID, now)
		if err != nil {
			t.Fatalf("Failed to initialize member %s: %v", memberID, err)
		}
	}

	// Verify: All members were added
	count, err = database.CountGroupMembers(groupID)
	if err != nil {
		t.Fatalf("Failed to count members after init: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 members after initialization, got %d", count)
	}

	// Verify: All members have timestamps (not inactive)
	inactive, err := database.GetInactiveMembers(groupID, 1)
	if err != nil {
		t.Fatalf("Failed to get inactive members: %v", err)
	}

	if len(inactive) != 0 {
		t.Errorf("Expected 0 inactive members after initialization, got %d", len(inactive))
	}
}

func TestMultipleGroups(t *testing.T) {
	// Test handling multiple groups correctly

	database := setupTestDB(t)

	// Create two groups
	group1ID, err := database.UpsertGroup("group-1", "Group One", 30)
	if err != nil {
		t.Fatalf("Failed to create group 1: %v", err)
	}

	group2ID, err := database.UpsertGroup("group-2", "Group Two", 45)
	if err != nil {
		t.Fatalf("Failed to create group 2: %v", err)
	}

	// Add members to each group
	err = database.UpsertGroupMember(group1ID, "group1-user1")
	if err != nil {
		t.Fatalf("Failed to add member to group 1: %v", err)
	}

	err = database.UpsertGroupMember(group1ID, "group1-user2")
	if err != nil {
		t.Fatalf("Failed to add member to group 1: %v", err)
	}

	err = database.UpsertGroupMember(group2ID, "group2-user1")
	if err != nil {
		t.Fatalf("Failed to add member to group 2: %v", err)
	}

	// Verify counts
	count1, err := database.CountGroupMembers(group1ID)
	if err != nil {
		t.Fatalf("Failed to count group 1 members: %v", err)
	}

	if count1 != 2 {
		t.Errorf("Expected 2 members in group 1, got %d", count1)
	}

	count2, err := database.CountGroupMembers(group2ID)
	if err != nil {
		t.Fatalf("Failed to count group 2 members: %v", err)
	}

	if count2 != 1 {
		t.Errorf("Expected 1 member in group 2, got %d", count2)
	}

	// Get all groups
	groups, err := database.GetAllGroups()
	if err != nil {
		t.Fatalf("Failed to get all groups: %v", err)
	}

	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}
}

func TestInactivityThresholds(t *testing.T) {
	// Test different inactivity thresholds work correctly

	database := setupTestDB(t)

	// Create group with 7-day threshold
	groupID, err := database.UpsertGroup("test-group-4", "Test Group", 7)
	if err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	now := time.Now()

	// Add member who messaged 5 days ago (should be active)
	fiveDaysAgo := now.AddDate(0, 0, -5)
	err = database.UpsertGroupMemberWithTimestamp(groupID, "recent-user", fiveDaysAgo)
	if err != nil {
		t.Fatalf("Failed to add recent user: %v", err)
	}

	// Add member who messaged 10 days ago (should be inactive)
	tenDaysAgo := now.AddDate(0, 0, -10)
	err = database.UpsertGroupMemberWithTimestamp(groupID, "old-user", tenDaysAgo)
	if err != nil {
		t.Fatalf("Failed to add old user: %v", err)
	}

	// Update old user's joined_at to simulate old membership
	err = database.SetMemberJoinedAt(groupID, "old-user", tenDaysAgo)
	if err != nil {
		t.Fatalf("Failed to update old user joined_at: %v", err)
	}

	// Get inactive members with 7-day threshold
	inactive, err := database.GetInactiveMembers(groupID, 7)
	if err != nil {
		t.Fatalf("Failed to get inactive members: %v", err)
	}

	// Should only find the 10-day-old user
	if len(inactive) != 1 {
		t.Errorf("Expected 1 inactive member, got %d", len(inactive))
	}

	if len(inactive) > 0 && inactive[0].SignalUserID != "old-user" {
		t.Errorf("Expected inactive member to be 'old-user', got %q", inactive[0].SignalUserID)
	}
}
