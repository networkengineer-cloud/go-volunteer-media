package groupme

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	// GroupMe API endpoint for posting bot messages
	defaultAPIURL = "https://api.groupme.com/v3/bots/post"
	
	// Maximum message length for GroupMe
	maxMessageLength = 1000
)

// Service represents a GroupMe messaging service
type Service struct {
	apiURL string
	client *http.Client
}

// NewService creates a new GroupMe service
func NewService() *Service {
	return &Service{
		apiURL: defaultAPIURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendMessage sends a message to a GroupMe bot
func (s *Service) SendMessage(botID, text string) error {
	if botID == "" {
		return fmt.Errorf("bot ID is required")
	}
	
	if text == "" {
		return fmt.Errorf("message text is required")
	}
	
	// Truncate message if too long
	if len(text) > maxMessageLength {
		text = truncateMessage(text, maxMessageLength)
	}
	
	// Create request payload
	payload := map[string]string{
		"bot_id": botID,
		"text":   text,
	}
	
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	
	// Create HTTP request
	req, err := http.NewRequest(http.MethodPost, s.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send GroupMe message: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GroupMe API error: status %d", resp.StatusCode)
	}
	
	return nil
}

// SendAnnouncement sends a formatted announcement to a GroupMe bot
func (s *Service) SendAnnouncement(botID, title, content string) error {
	if title == "" {
		return fmt.Errorf("title is required")
	}
	
	if content == "" {
		return fmt.Errorf("content is required")
	}
	
	// Format the message
	message := formatAnnouncementMessage(title, content)
	
	return s.SendMessage(botID, message)
}

// formatAnnouncementMessage formats an announcement for GroupMe
func formatAnnouncementMessage(title, content string) string {
	return fmt.Sprintf("📢 %s\n\n%s", title, content)
}

// truncateMessage truncates a message to the specified length
func truncateMessage(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	
	const ellipsis = "..."
	if maxLength <= len(ellipsis) {
		return text[:maxLength]
	}
	
	return text[:maxLength-len(ellipsis)] + ellipsis
}
