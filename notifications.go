package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

// NtfyMessage represents a message to be sent to the ntfy.sh service
type NtfyMessage struct {
	Message  string `json:"message"`
	Priority string `json:"priority,omitempty"`
}

// sendNtfyNotification sends a notification to the ntfy.sh service with the specified topic
func sendNtfyNotification(settings *Settings, message string, notificationTopic string) error {
	if settings.NtfyTopic == nil || *settings.NtfyTopic == "" {
		return nil
	}

	topic := *settings.NtfyTopic
	if notificationTopic == "warning" && settings.NtfyTopicWarning != nil {
		topic = *settings.NtfyTopicWarning
	}

	url := fmt.Sprintf("%s/%s", settings.NtfyServer, topic)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte(message)))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("notification failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendNotification sends a notification through the specified notification channels
func sendNotification(settings *Settings, message string, notificationTopic string, notificationTypes []string) error {
	for _, nt := range notificationTypes {
		switch NotificationType(nt) {
		case NotificationTypeNtfy:
			if err := sendNtfyNotification(settings, message, notificationTopic); err != nil {
				return fmt.Errorf("error sending ntfy notification: %w", err)
			}
		}
	}
	return nil
}
