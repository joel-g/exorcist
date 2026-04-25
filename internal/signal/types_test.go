package signal

import (
	"testing"
	"time"
)

func TestMessage_GetTimestamp(t *testing.T) {
	tests := []struct {
		name      string
		message   Message
		wantTime  int64 // Unix timestamp in milliseconds
	}{
		{
			name: "uses envelope timestamp when data message is nil",
			message: Message{
				Envelope: struct {
					Source       string `json:"source"`
					SourceNumber string `json:"sourceNumber"`
					SourceUUID   string `json:"sourceUuid"`
					SourceName   string `json:"sourceName"`
					Timestamp    int64  `json:"timestamp"`
					DataMessage  *struct {
						Timestamp int64  `json:"timestamp"`
						Message   string `json:"message"`
						GroupInfo *struct {
							GroupID string `json:"groupId"`
							Type    string `json:"type"`
						} `json:"groupInfo"`
					} `json:"dataMessage"`
				}{
					Timestamp: 1640000000000, // Jan 1, 2022
				},
			},
			wantTime: 1640000000000,
		},
		{
			name: "uses data message timestamp when available",
			message: Message{
				Envelope: struct {
					Source       string `json:"source"`
					SourceNumber string `json:"sourceNumber"`
					SourceUUID   string `json:"sourceUuid"`
					SourceName   string `json:"sourceName"`
					Timestamp    int64  `json:"timestamp"`
					DataMessage  *struct {
						Timestamp int64  `json:"timestamp"`
						Message   string `json:"message"`
						GroupInfo *struct {
							GroupID string `json:"groupId"`
							Type    string `json:"type"`
						} `json:"groupInfo"`
					} `json:"dataMessage"`
				}{
					Timestamp: 1640000000000,
					DataMessage: &struct {
						Timestamp int64  `json:"timestamp"`
						Message   string `json:"message"`
						GroupInfo *struct {
							GroupID string `json:"groupId"`
							Type    string `json:"type"`
						} `json:"groupInfo"`
					}{
						Timestamp: 1650000000000, // Newer timestamp
					},
				},
			},
			wantTime: 1650000000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.message.GetTimestamp()
			want := time.Unix(tt.wantTime/1000, 0)

			// Allow 1 second difference for rounding
			if got.Sub(want).Abs() > time.Second {
				t.Errorf("GetTimestamp() = %v, want %v", got, want)
			}
		})
	}
}

func TestMessage_GetGroupID(t *testing.T) {
	tests := []struct {
		name    string
		message Message
		want    string
	}{
		{
			name: "returns empty string when no group info",
			message: Message{
				Envelope: struct {
					Source       string `json:"source"`
					SourceNumber string `json:"sourceNumber"`
					SourceUUID   string `json:"sourceUuid"`
					SourceName   string `json:"sourceName"`
					Timestamp    int64  `json:"timestamp"`
					DataMessage  *struct {
						Timestamp int64  `json:"timestamp"`
						Message   string `json:"message"`
						GroupInfo *struct {
							GroupID string `json:"groupId"`
							Type    string `json:"type"`
						} `json:"groupInfo"`
					} `json:"dataMessage"`
				}{},
			},
			want: "",
		},
		{
			name: "returns group ID when available",
			message: Message{
				Envelope: struct {
					Source       string `json:"source"`
					SourceNumber string `json:"sourceNumber"`
					SourceUUID   string `json:"sourceUuid"`
					SourceName   string `json:"sourceName"`
					Timestamp    int64  `json:"timestamp"`
					DataMessage  *struct {
						Timestamp int64  `json:"timestamp"`
						Message   string `json:"message"`
						GroupInfo *struct {
							GroupID string `json:"groupId"`
							Type    string `json:"type"`
						} `json:"groupInfo"`
					} `json:"dataMessage"`
				}{
					DataMessage: &struct {
						Timestamp int64  `json:"timestamp"`
						Message   string `json:"message"`
						GroupInfo *struct {
							GroupID string `json:"groupId"`
							Type    string `json:"type"`
						} `json:"groupInfo"`
					}{
						GroupInfo: &struct {
							GroupID string `json:"groupId"`
							Type    string `json:"type"`
						}{
							GroupID: "test-group-123",
						},
					},
				},
			},
			want: "test-group-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.message.GetGroupID(); got != tt.want {
				t.Errorf("GetGroupID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessage_GetSenderID(t *testing.T) {
	tests := []struct {
		name    string
		message Message
		want    string
	}{
		{
			name: "returns UUID when available",
			message: Message{
				Envelope: struct {
					Source       string `json:"source"`
					SourceNumber string `json:"sourceNumber"`
					SourceUUID   string `json:"sourceUuid"`
					SourceName   string `json:"sourceName"`
					Timestamp    int64  `json:"timestamp"`
					DataMessage  *struct {
						Timestamp int64  `json:"timestamp"`
						Message   string `json:"message"`
						GroupInfo *struct {
							GroupID string `json:"groupId"`
							Type    string `json:"type"`
						} `json:"groupInfo"`
					} `json:"dataMessage"`
				}{
					SourceUUID:   "uuid-123",
					SourceNumber: "+1234567890",
				},
			},
			want: "uuid-123",
		},
		{
			name: "returns phone number when UUID is empty",
			message: Message{
				Envelope: struct {
					Source       string `json:"source"`
					SourceNumber string `json:"sourceNumber"`
					SourceUUID   string `json:"sourceUuid"`
					SourceName   string `json:"sourceName"`
					Timestamp    int64  `json:"timestamp"`
					DataMessage  *struct {
						Timestamp int64  `json:"timestamp"`
						Message   string `json:"message"`
						GroupInfo *struct {
							GroupID string `json:"groupId"`
							Type    string `json:"type"`
						} `json:"groupInfo"`
					} `json:"dataMessage"`
				}{
					SourceUUID:   "",
					SourceNumber: "+1234567890",
				},
			},
			want: "+1234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.message.GetSenderID(); got != tt.want {
				t.Errorf("GetSenderID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	account := "+1234567890"
	client := NewClient(account)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.account != account {
		t.Errorf("Expected account %q, got %q", account, client.account)
	}
}
