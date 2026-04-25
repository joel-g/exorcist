package jobs

import (
	"fmt"
	"log"
	"time"

	"github.com/joel-g/exorcist/internal/db"
	"github.com/joel-g/exorcist/internal/signal"
)

// InitJob handles initializing the database with existing group members
type InitJob struct {
	db     *db.DB
	signal *signal.Client
}

// NewInitJob creates a new initialization job
func NewInitJob(database *db.DB, signalClient *signal.Client) *InitJob {
	return &InitJob{
		db:     database,
		signal: signalClient,
	}
}

// Run executes the initialization job
// This sets all existing group members' last_message_at to NOW()
// so the inactivity countdown starts from when monitoring begins
func (j *InitJob) Run() error {
	log.Println("Starting initialization job - seeding existing group members...")

	// Get all monitored groups from database
	groups, err := j.db.GetAllGroups()
	if err != nil {
		return fmt.Errorf("failed to get groups: %w", err)
	}

	if len(groups) == 0 {
		log.Println("No groups configured for monitoring")
		return nil
	}

	totalAdded := 0
	now := time.Now()

	// Process each group
	for _, group := range groups {
		log.Printf("Initializing members for group: %s (ID: %d)", group.Name, group.ID)

		// Fetch members from Signal
		members, err := j.signal.GetGroupMembers(group.SignalGroupID)
		if err != nil {
			log.Printf("Failed to get members for group %s: %v", group.Name, err)
			continue
		}

		log.Printf("Found %d members in group %s", len(members), group.Name)

		// Add each member with current timestamp
		// PRIVACY: Only stores signal_user_id and timestamp
		for _, memberID := range members {
			err := j.db.UpsertGroupMemberWithTimestamp(group.ID, memberID, now)
			if err != nil {
				log.Printf("Failed to add member %s to group %d: %v", memberID, group.ID, err)
				continue
			}

			totalAdded++
			log.Printf("Added member %s to group %s with timestamp NOW()", memberID, group.Name)
		}
	}

	log.Printf("Initialization complete. Total members added: %d", totalAdded)
	log.Println("Inactivity countdown starts now - members who don't message in the next N days will be removed")

	return nil
}
