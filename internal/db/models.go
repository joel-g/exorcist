package db

import "time"

// Group represents a Signal group chat being monitored
type Group struct {
	ID            int64     `db:"id"`
	SignalGroupID string    `db:"signal_group_id"`
	Name          string    `db:"name"`
	InactivityDays int      `db:"inactivity_days"`
	CreatedAt     time.Time `db:"created_at"`
}

// GroupMember represents a user's membership in a group
// PRIVACY: Only stores signal_user_id (required for API) and timestamps
type GroupMember struct {
	ID            int64      `db:"id"`
	GroupID       int64      `db:"group_id"`
	SignalUserID  string     `db:"signal_user_id"`
	LastMessageAt *time.Time `db:"last_message_at"`
	JoinedAt      time.Time  `db:"joined_at"`
	CreatedAt     time.Time  `db:"created_at"`
}
