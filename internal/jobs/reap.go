package jobs

import (
	"fmt"
	"log"
	"time"

	"github.com/joel-g/exorcist/internal/db"
	"github.com/joel-g/exorcist/internal/signal"
)

// ReapJob handles removing inactive users from groups
type ReapJob struct {
	db     *db.DB
	signal *signal.Client
}

// NewReapJob creates a new reap job
func NewReapJob(database *db.DB, signalClient *signal.Client) *ReapJob {
	return &ReapJob{
		db:     database,
		signal: signalClient,
	}
}

// Run executes the reap job
func (j *ReapJob) Run() error {
	log.Println("Starting ghost reaping job...")

	// Get all monitored groups
	groups, err := j.db.GetAllGroups()
	if err != nil {
		return fmt.Errorf("failed to get groups: %w", err)
	}

	totalReaped := 0

	// Process each group
	for _, group := range groups {
		log.Printf("Checking group: %s (ID: %d, inactivity threshold: %d days)",
			group.Name, group.ID, group.InactivityDays)

		// Get inactive members for this group
		inactiveMembers, err := j.db.GetInactiveMembers(group.ID, group.InactivityDays)
		if err != nil {
			log.Printf("Failed to get inactive members for group %d: %v", group.ID, err)
			continue
		}

		if len(inactiveMembers) == 0 {
			log.Printf("No inactive members in group %s", group.Name)
			continue
		}

		log.Printf("Found %d inactive members in group %s", len(inactiveMembers), group.Name)

		// Remove each inactive member
		for _, member := range inactiveMembers {
			log.Printf("Removing ghost: %s from group %s", member.SignalUserID, group.Name)

			// Build announcement message
			// Signal will show who was removed, so we just explain why
			var lastActiveMsg string
			if member.LastMessageAt != nil {
				lastActiveMsg = fmt.Sprintf("last activity %s", member.LastMessageAt.Format("January 2, 2006"))
			} else {
				lastActiveMsg = "no activity recorded"
			}

			announcement := fmt.Sprintf("🤖 Automated removal - %s - inactive for %d+ days.",
				lastActiveMsg, group.InactivityDays)

			// Send announcement to group
			if err := j.signal.SendGroupMessage(group.SignalGroupID, announcement); err != nil {
				log.Printf("Failed to send announcement for %s: %v", member.SignalUserID, err)
				// Continue anyway - announcement failure shouldn't block removal
			} else {
				log.Printf("Sent removal announcement for %s", member.SignalUserID)
				// Brief pause to ensure message is delivered before removal
				time.Sleep(2 * time.Second)
			}

			// Remove from Signal group
			err := j.signal.RemoveGroupMember(group.SignalGroupID, member.SignalUserID)
			if err != nil {
				log.Printf("Failed to remove member %s from Signal group: %v",
					member.SignalUserID, err)
				continue
			}

			// Remove from database
			err = j.db.RemoveMember(group.ID, member.SignalUserID)
			if err != nil {
				log.Printf("Failed to remove member %s from database: %v",
					member.SignalUserID, err)
				continue
			}

			totalReaped++
			log.Printf("Successfully removed ghost: %s from %s", member.SignalUserID, group.Name)
		}
	}

	log.Printf("Ghost reaping job completed. Total ghosts removed: %d", totalReaped)
	return nil
}
