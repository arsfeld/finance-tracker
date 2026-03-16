package api

import (
	"encoding/json"
	"net/http"

	"finance_tracker/internal/store"
)

type AccountHandler struct {
	store *store.AccountStore
}

func NewAccountHandler(s *store.AccountStore) *AccountHandler {
	return &AccountHandler{store: s}
}

func (h *AccountHandler) List(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.store.List(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	WriteData(w, accounts)
}

func (h *AccountHandler) UpdateInclusion(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		IsIncluded bool `json:"is_included"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body")
		return
	}

	if err := h.store.UpdateInclusion(r.Context(), id, body.IsIncluded); err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	WriteData(w, map[string]string{"status": "ok"})
}
