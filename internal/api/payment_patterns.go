package api

import (
	"encoding/json"
	"net/http"

	"finance_tracker/internal/ledger"
	"finance_tracker/internal/store"
)

// PaymentPatternsHandler exposes the descriptions that identify payments toward
// each card from the paying accounts. The settings page only reflects .env, so
// these get their own endpoint.
type PaymentPatternsHandler struct {
	settings *store.SettingsStore
}

func NewPaymentPatternsHandler(s *store.SettingsStore) *PaymentPatternsHandler {
	return &PaymentPatternsHandler{settings: s}
}

func (h *PaymentPatternsHandler) Get(w http.ResponseWriter, r *http.Request) {
	raw, err := h.settings.Get(r.Context(), ledger.PaymentPatternsSettingKey)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	patterns, err := ledger.ParsePaymentPatterns(raw)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "BAD_SETTING", err.Error())
		return
	}
	WriteData(w, patterns)
}

// Put replaces the whole map, keyed by card key. JSON null decodes into a nil
// map without error, and storing it would silently wipe every card's patterns,
// so it is rejected with the malformed bodies. An empty object still clears
// them on purpose.
func (h *PaymentPatternsHandler) Put(w http.ResponseWriter, r *http.Request) {
	var patterns map[string][]string
	if err := json.NewDecoder(r.Body).Decode(&patterns); err != nil || patterns == nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Expected a JSON object of card key to pattern list")
		return
	}
	raw, _ := json.Marshal(patterns)
	if err := h.settings.Set(r.Context(), ledger.PaymentPatternsSettingKey, string(raw)); err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	WriteData(w, patterns)
}
