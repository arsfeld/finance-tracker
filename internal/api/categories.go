package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"finance_tracker/internal/store"
)

type CategoryHandler struct {
	catStore *store.CategoryStore
	txnStore *store.TransactionStore
	events   *EventHub
}

func NewCategoryHandler(cs *store.CategoryStore, ts *store.TransactionStore, events *EventHub) *CategoryHandler {
	return &CategoryHandler{catStore: cs, txnStore: ts, events: events}
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	categories, err := h.catStore.ListAll(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	WriteData(w, categories)
}

// ListUnique returns distinct category names with exclusion status and merchant count.
func (h *CategoryHandler) ListUnique(w http.ResponseWriter, r *http.Request) {
	cats, err := h.catStore.ListUniqueCategories(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if cats == nil {
		cats = []store.CategoryInfo{}
	}
	WriteData(w, cats)
}

// SetExcluded toggles whether a category is excluded from analysis.
func (h *CategoryHandler) SetExcluded(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Category string `json:"category"`
		Excluded bool   `json:"excluded"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON")
		return
	}
	if body.Category == "" {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Category name required")
		return
	}

	if err := h.catStore.SetCategoryExcluded(r.Context(), body.Category, body.Excluded); err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	h.events.Broadcast("categories_updated", `{"status":"ok"}`)
	WriteData(w, map[string]string{"status": "ok"})
}

func (h *CategoryHandler) Summary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	start, _ := strconv.ParseInt(q.Get("start"), 10, 64)
	end, _ := strconv.ParseInt(q.Get("end"), 10, 64)

	if start == 0 || end == 0 {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "start and end query params required (unix timestamps)")
		return
	}

	totals, err := h.txnStore.CountByCategory(r.Context(), start, end)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	WriteData(w, totals)
}
