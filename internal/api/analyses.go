package api

import (
	"net/http"
	"strconv"

	"finance_tracker/internal/store"
)

type AnalysisHandler struct {
	store *store.AnalysisStore
}

func NewAnalysisHandler(s *store.AnalysisStore) *AnalysisHandler {
	return &AnalysisHandler{store: s}
}

func (h *AnalysisHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	analyses, err := h.store.List(r.Context(), limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	WriteData(w, analyses)
}

func (h *AnalysisHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid analysis ID")
		return
	}

	analysis, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if analysis == nil {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "Analysis not found")
		return
	}
	WriteData(w, analysis)
}

func (h *AnalysisHandler) GetLatest(w http.ResponseWriter, r *http.Request) {
	analysis, err := h.store.GetLatest(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if analysis == nil {
		WriteData(w, nil)
		return
	}
	WriteData(w, analysis)
}
