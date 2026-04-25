package db

// Schema defines the database structure
// PRIVACY: No message content is ever stored, only timestamps
const Schema = `
CREATE TABLE IF NOT EXISTS groups (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	signal_group_id TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	inactivity_days INTEGER NOT NULL DEFAULT 30,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS group_members (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	group_id INTEGER NOT NULL,
	signal_user_id TEXT NOT NULL,  -- UUID or phone number from Signal (required for API calls)
	last_message_at TIMESTAMP,     -- Only timestamp stored, never message content
	joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
	UNIQUE(group_id, signal_user_id)
);

CREATE INDEX IF NOT EXISTS idx_group_members_last_message
	ON group_members(group_id, last_message_at);
CREATE INDEX IF NOT EXISTS idx_groups_signal_id
	ON groups(signal_group_id);
`
