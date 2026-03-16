package api

import (
	"net/http"
	"strconv"

	"finance_tracker/internal/store"
)

type CategoryHandler struct {
	catStore *store.CategoryStore
	txnStore *store.TransactionStore
}

func NewCategoryHandler(cs *store.CategoryStore, ts *store.TransactionStore) *CategoryHandler {
	return &CategoryHandler{catStore: cs, txnStore: ts}
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	categories, err := h.catStore.ListAll(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	WriteData(w, categories)
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
