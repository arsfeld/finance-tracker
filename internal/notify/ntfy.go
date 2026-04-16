package notify

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/rs/zerolog/log"
)

// SendNtfy sends a notification to the ntfy.sh service.
func SendNtfy(server, topic, warningSuffix, message, notificationType string) error {
	if topic == "" {
		return nil
	}

	finalTopic := topic
	if notificationType == "warning" {
		finalTopic = topic + warningSuffix
	}

	plainMessage := stripMarkdown(message)

	url := fmt.Sprintf("%s/%s", server, finalTopic)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte(plainMessage)))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Title", "Finance Tracker")

	if user := os.Getenv("NTFY_PUBLISHER_USER"); user != "" {
		if pass := os.Getenv("NTFY_PUBLISHER_PASS"); pass != "" {
			req.SetBasicAuth(user, pass)
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("notification failed with status %d: %s", resp.StatusCode, string(body))
	}

	log.Info().Str("topic", finalTopic).Msg("Ntfy notification sent")
	return nil
}

func stripMarkdown(text string) string {
	text = regexp.MustCompile(`\*\*(.*?)\*\*|__(.*?)__`).ReplaceAllString(text, "$1$2")
	text = regexp.MustCompile(`\*(.*?)\*|_(.*?)_`).ReplaceAllString(text, "$1$2")
	text = regexp.MustCompile(`^#+\s+`).ReplaceAllString(text, "")
	return text
}
