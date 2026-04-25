package jobs

import (
	"fmt"
	"log"

	"github.com/joel-g/exorcist/internal/db"
	"github.com/joel-g/exorcist/internal/signal"
)

// SyncJob handles syncing messages from Signal and updating last_message_at
type SyncJob struct {
	db     *db.DB
	signal *signal.Client
}

// NewSyncJob creates a new sync job
func NewSyncJob(database *db.DB, signalClient *signal.Client) *SyncJob {
	return &SyncJob{
		db:     database,
		signal: signalClient,
	}
}

// Run executes the sync job
func (j *SyncJob) Run() error {
	log.Println("Starting message sync job...")

	// Receive new messages from Signal
	messages, err := j.signal.ReceiveMessages()
	if err != nil {
		return fmt.Errorf("failed to receive messages: %w", err)
	}

	log.Printf("Received %d messages", len(messages))

	// Get all monitored groups from database
	groups, err := j.db.GetAllGroups()
	if err != nil {
		return fmt.Errorf("failed to get groups: %w", err)
	}

	// Create map of signal_group_id -> db group_id for quick lookup
	groupMap := make(map[string]int64)
	for _, g := range groups {
		groupMap[g.SignalGroupID] = g.ID
	}

	// Process each message
	// PRIVACY: We only extract timestamps, never store message content
	for _, msg := range messages {
		groupID := msg.GetGroupID()
		if groupID == "" {
			// Skip non-group messages
			continue
		}

		dbGroupID, exists := groupMap[groupID]
		if !exists {
			// This group is not being monitored, skip
			continue
		}

		senderID := msg.GetSenderID()
		timestamp := msg.GetTimestamp()

		// Upsert member and update last_message_at (auto-adds new members)
		// PRIVACY: Only stores signal_user_id and timestamp, never message content
		if err := j.db.UpsertGroupMemberWithTimestamp(dbGroupID, senderID, timestamp); err != nil {
			log.Printf("Failed to upsert member %s in group %d: %v",
				senderID, dbGroupID, err)
			continue
		}

		log.Printf("Updated last message for user %s in group %d", senderID, dbGroupID)
	}

	log.Println("Message sync job completed")
	return nil
}
