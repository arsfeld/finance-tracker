package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"finance_tracker/internal/models"
	"finance_tracker/internal/store"
)

type FilterHandler struct {
	store *store.FilterStore
}

func NewFilterHandler(s *store.FilterStore) *FilterHandler {
	return &FilterHandler{store: s}
}

func (h *FilterHandler) List(w http.ResponseWriter, r *http.Request) {
	rules, err := h.store.List(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if rules == nil {
		rules = []models.FilterRule{}
	}
	WriteData(w, rules)
}

func (h *FilterHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Pattern   string           `json:"pattern"`
		MatchType models.MatchType `json:"match_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body")
		return
	}

	if body.Pattern == "" {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Pattern is required")
		return
	}
	if body.MatchType == "" {
		body.MatchType = models.MatchTypeSubstring
	}

	id, err := h.store.Create(r.Context(), body.Pattern, body.MatchType)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	WriteJSON(w, http.StatusCreated, Response{Data: map[string]int64{"id": id}})
}

func (h *FilterHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid filter ID")
		return
	}

	var body struct {
		Pattern   string           `json:"pattern"`
		MatchType models.MatchType `json:"match_type"`
		IsActive  bool             `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body")
		return
	}

	if err := h.store.Update(r.Context(), id, body.Pattern, body.MatchType, body.IsActive); err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	WriteData(w, map[string]string{"status": "ok"})
}

func (h *FilterHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid filter ID")
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	WriteData(w, map[string]string{"status": "ok"})
}
