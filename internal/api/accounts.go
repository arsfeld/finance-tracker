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

func (h *AccountHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var patch store.AccountPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body")
		return
	}
	if patch.CardKey != nil && *patch.CardKey == "" {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "card_key cannot be empty")
		return
	}

	existing, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if existing == nil {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "Account not found")
		return
	}

	if err := h.store.Update(r.Context(), id, patch); err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	WriteData(w, map[string]string{"status": "ok"})
}
