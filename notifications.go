package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type NtfyMessage struct {
	Message  string `json:"message"`
	Priority string `json:"priority,omitempty"`
}

func sendNtfyNotification(settings *Settings, message string, notificationType string) error {
	if settings.NtfyTopic == nil || *settings.NtfyTopic == "" {
		return nil
	}

	priority := "default"
	if notificationType == "warning" {
		priority = "high"
	}

	ntfyMsg := NtfyMessage{
		Message:  message,
		Priority: priority,
	}

	jsonData, err := json.Marshal(ntfyMsg)
	if err != nil {
		return fmt.Errorf("error marshaling ntfy message: %w", err)
	}

	url := fmt.Sprintf("%s/%s", settings.NtfyServer, *settings.NtfyTopic)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
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

func sendNotification(settings *Settings, message string, notificationType string, notificationTypes []string) error {
	for _, nt := range notificationTypes {
		switch nt {
		case string(NotificationTypeNtfy):
			if err := sendNtfyNotification(settings, message, notificationType); err != nil {
				return fmt.Errorf("error sending ntfy notification: %w", err)
			}
		// Add other notification types here (SMS, email)
		}
	}
	return nil
} 