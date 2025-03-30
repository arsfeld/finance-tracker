package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

type NtfyMessage struct {
	Message  string `json:"message"`
	Priority string `json:"priority,omitempty"`
}

func sendNtfyNotification(settings *Settings, message string, notificationTopic string) error {
	if settings.NtfyTopic == nil || *settings.NtfyTopic == "" {
		return nil
	}

	switch notificationTopic {
	case "info":
		notificationTopic = *settings.NtfyTopic
	case "warning":
		notificationTopic = *settings.NtfyTopicWarning
	}

	url := fmt.Sprintf("%s/%s", settings.NtfyServer, notificationTopic)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(message)))
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

func sendNotification(settings *Settings, message string, notificationTopic string, notificationTypes []string) error {
	for _, nt := range notificationTypes {
		switch nt {
		case string(NotificationTypeNtfy):
			if err := sendNtfyNotification(settings, message, notificationTopic); err != nil {
				return fmt.Errorf("error sending ntfy notification: %w", err)
			}
		}
	}
	return nil
}
