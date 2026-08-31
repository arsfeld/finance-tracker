package api

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"

	"finance_tracker/internal/config"
	"finance_tracker/internal/notify"
)

// Response wraps API data responses.
type Response struct {
	Data interface{} `json:"data"`
	Meta *Meta       `json:"meta,omitempty"`
}

// Meta holds pagination metadata.
type Meta struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

// ErrorBody wraps API error responses.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error code and message.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Error().Err(err).Msg("Failed to write JSON response")
	}
}

// WriteData writes a successful JSON data response.
func WriteData(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusOK, Response{Data: data})
}

// WriteDataWithMeta writes a successful JSON data response with pagination metadata.
func WriteDataWithMeta(w http.ResponseWriter, data interface{}, meta *Meta) {
	WriteJSON(w, http.StatusOK, Response{Data: data, Meta: meta})
}

// WriteError writes a JSON error response.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorBody{
		Error: ErrorDetail{Code: code, Message: message},
	})
}

// newDispatcher builds a notification dispatcher from the app config. Sync
// alerts and analysis reports go out the same channels, so the wiring lives in
// one place.
func newDispatcher(cfg *config.Config) *notify.Dispatcher {
	return notify.NewDispatcher(notify.Config{
		NtfyServer:        cfg.NtfyServer,
		NtfyTopic:         cfg.NtfyTopic,
		NtfyWarningSuffix: cfg.NtfyWarningSuffix,
		MailerURL:         cfg.MailerURL,
		MailerFrom:        cfg.MailerFrom,
		MailerTo:          cfg.MailerTo,
	})
}
