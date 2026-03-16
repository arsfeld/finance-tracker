package api

import (
	"encoding/json"
	"net/http"

	"finance_tracker/internal/config"
	"finance_tracker/internal/notify"
	"finance_tracker/internal/store"
)

type SettingsHandler struct {
	store *store.SettingsStore
	cfg   *config.Config
}

func NewSettingsHandler(s *store.SettingsStore, cfg *config.Config) *SettingsHandler {
	return &SettingsHandler{store: s, cfg: cfg}
}

func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	settings, err := h.store.GetAll(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	WriteData(w, settings)
}

func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body")
		return
	}

	if err := h.store.SetMultiple(r.Context(), body); err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	WriteData(w, map[string]string{"status": "ok"})
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
