package signal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Client wraps signal-cli commands
type Client struct {
	account string // Phone number of the signal account
}

// NewClient creates a new Signal client
func NewClient(account string) *Client {
	return &Client{account: account}
}

// execCommand runs a signal-cli command and returns the output
func (c *Client) execCommand(args ...string) ([]byte, error) {
	fullArgs := append([]string{"-a", c.account}, args...)
	cmd := exec.Command("signal-cli", fullArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("signal-cli error: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// ListGroups returns all groups the account is a member of
func (c *Client) ListGroups() ([]Group, error) {
	output, err := c.execCommand("listGroups", "--detailed")
	if err != nil {
		return nil, err
	}

	var groups []Group
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "Id:") {
			continue
		}

		// Parse group info (basic parsing, signal-cli output format may vary)
		var group Group
		if strings.Contains(line, "Id:") {
			parts := strings.Split(line, "Id:")
			if len(parts) > 1 {
				idPart := strings.TrimSpace(parts[1])
				group.ID = strings.Fields(idPart)[0]
			}
		}

		if group.ID != "" {
			groups = append(groups, group)
		}
	}

	return groups, nil
}

// ReceiveMessages receives messages for the account
// Returns messages since the last receive call
func (c *Client) ReceiveMessages() ([]Message, error) {
	output, err := c.execCommand("receive", "--json")
	if err != nil {
		return nil, err
	}

	var messages []Message
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			// Skip malformed messages
			continue
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

// SendGroupMessage sends a message to a group
func (c *Client) SendGroupMessage(groupID, message string) error {
	_, err := c.execCommand("send", "-g", groupID, "-m", message)
	return err
}

// RemoveGroupMember removes a member from a group
func (c *Client) RemoveGroupMember(groupID, memberID string) error {
	_, err := c.execCommand("quitGroup", "--group-id", groupID, "--member", memberID)
	return err
}

// GetGroupInfo returns detailed information about a group
func (c *Client) GetGroupInfo(groupID string) (*Group, error) {
	// Note: signal-cli doesn't have a direct way to get single group info
	// We'll list all groups and filter
	groups, err := c.ListGroups()
	if err != nil {
		return nil, err
	}

	for _, g := range groups {
		if g.ID == groupID {
			return &g, nil
		}
	}

	return nil, fmt.Errorf("group not found: %s", groupID)
}

// GetGroupMembers returns all members of a group
func (c *Client) GetGroupMembers(groupID string) ([]string, error) {
	// Use listMembers command to get group members
	output, err := c.execCommand("listMembers", "-g", groupID)
	if err != nil {
		return nil, err
	}

	var members []string
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Each line should be a member ID (UUID or phone number)
		members = append(members, line)
	}

	return members, nil
}
