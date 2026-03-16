package notify

import (
	"fmt"

	"github.com/rs/zerolog/log"

	"finance_tracker/internal/models"
)

// Config holds notification settings.
type Config struct {
	NtfyServer        string
	NtfyTopic         string
	NtfyWarningSuffix string
	MailerURL         string
	MailerFrom        string
	MailerTo          string
}

// Dispatcher sends notifications through configured channels.
type Dispatcher struct {
	cfg Config
}

// NewDispatcher creates a new notification dispatcher.
func NewDispatcher(cfg Config) *Dispatcher {
	return &Dispatcher{cfg: cfg}
}

// Send dispatches a notification through all configured channels.
func (d *Dispatcher) Send(message string, transactions []models.DBTransaction, notificationType string) ([]string, error) {
	var channels []string

	// Ntfy
	if d.cfg.NtfyTopic != "" {
		if err := SendNtfy(d.cfg.NtfyServer, d.cfg.NtfyTopic, d.cfg.NtfyWarningSuffix, message, notificationType); err != nil {
			log.Error().Err(err).Msg("Failed to send ntfy notification")
			return channels, fmt.Errorf("ntfy: %w", err)
		}
		channels = append(channels, fmt.Sprintf("Ntfy: %s", d.cfg.NtfyTopic))
	}

	// Email
	if d.cfg.MailerURL != "" && d.cfg.MailerTo != "" {
		if err := SendEmail(d.cfg.MailerURL, d.cfg.MailerFrom, d.cfg.MailerTo, message, transactions); err != nil {
			log.Error().Err(err).Msg("Failed to send email notification")
			return channels, fmt.Errorf("email: %w", err)
		}
		channels = append(channels, fmt.Sprintf("Email: %s", d.cfg.MailerTo))
	}

	return channels, nil
}
