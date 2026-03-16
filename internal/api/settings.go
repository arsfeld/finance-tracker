package api

import (
	"net/http"

	"finance_tracker/internal/config"
	"finance_tracker/internal/notify"
)

type SettingsHandler struct {
	cfg *config.Config
}

func NewSettingsHandler(cfg *config.Config) *SettingsHandler {
	return &SettingsHandler{cfg: cfg}
}

// Get returns the current configuration (loaded from .env).
// Secrets are masked.
func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	WriteData(w, map[string]interface{}{
		"billing_day":           h.cfg.BillingDay,
		"sync_schedule":         h.cfg.SyncSchedule,
		"listen_addr":           h.cfg.ListenAddr,
		"simplefin_configured":  h.cfg.SimplefinBridgeURL != "",
		"openrouter_url":        h.cfg.OpenRouterURL,
		"openrouter_model":      h.cfg.OpenRouterModel,
		"openrouter_configured": h.cfg.OpenRouterAPIKey != "",
		"ntfy_server":           h.cfg.NtfyServer,
		"ntfy_topic":            h.cfg.NtfyTopic,
		"ntfy_warning_suffix":   h.cfg.NtfyWarningSuffix,
		"mailer_configured":     h.cfg.MailerURL != "",
		"mailer_from":           h.cfg.MailerFrom,
		"mailer_to":             h.cfg.MailerTo,
	})
}

func (h *SettingsHandler) TestNotification(w http.ResponseWriter, r *http.Request) {
	dispatcher := notify.NewDispatcher(notify.Config{
		NtfyServer:        h.cfg.NtfyServer,
		NtfyTopic:         h.cfg.NtfyTopic,
		NtfyWarningSuffix: h.cfg.NtfyWarningSuffix,
		MailerURL:         h.cfg.MailerURL,
		MailerFrom:        h.cfg.MailerFrom,
		MailerTo:          h.cfg.MailerTo,
	})

	channels, err := dispatcher.Send("Test notification from Finance Tracker web UI", nil, "info")
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "NOTIFICATION_ERROR", err.Error())
		return
	}

	WriteData(w, map[string]interface{}{"status": "ok", "channels": channels})
}
