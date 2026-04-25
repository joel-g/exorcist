package signal

import "time"

// Message represents a Signal message
type Message struct {
	Envelope struct {
		Source         string `json:"source"`
		SourceNumber   string `json:"sourceNumber"`
		SourceUUID     string `json:"sourceUuid"`
		SourceName     string `json:"sourceName"`
		Timestamp      int64  `json:"timestamp"`
		DataMessage    *struct {
			Timestamp int64  `json:"timestamp"`
			Message   string `json:"message"`
			GroupInfo *struct {
				GroupID string `json:"groupId"`
				Type    string `json:"type"`
			} `json:"groupInfo"`
		} `json:"dataMessage"`
	} `json:"envelope"`
}

// Group represents a Signal group
type Group struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Members     []string `json:"members"`
	IsMember    bool     `json:"isMember"`
	IsBlocked   bool     `json:"isBlocked"`
}

// GetTimestamp returns the message timestamp as time.Time
func (m *Message) GetTimestamp() time.Time {
	ts := m.Envelope.Timestamp
	if m.Envelope.DataMessage != nil && m.Envelope.DataMessage.Timestamp > 0 {
		ts = m.Envelope.DataMessage.Timestamp
	}
	return time.Unix(ts/1000, 0)
}

// GetGroupID returns the group ID if this is a group message
func (m *Message) GetGroupID() string {
	if m.Envelope.DataMessage != nil && m.Envelope.DataMessage.GroupInfo != nil {
		return m.Envelope.DataMessage.GroupInfo.GroupID
	}
	return ""
}

// GetSenderID returns the sender's UUID or phone number
func (m *Message) GetSenderID() string {
	if m.Envelope.SourceUUID != "" {
		return m.Envelope.SourceUUID
	}
	return m.Envelope.SourceNumber
}
