package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"finance_tracker/internal/store"
)

type TransactionHandler struct {
	store    *store.TransactionStore
	catStore *store.CategoryStore
	events   *EventHub
}

func NewTransactionHandler(s *store.TransactionStore, cs *store.CategoryStore, events *EventHub) *TransactionHandler {
	return &TransactionHandler{store: s, catStore: cs, events: events}
}

func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	startDate, _ := strconv.ParseInt(q.Get("start"), 10, 64)
	endDate, _ := strconv.ParseInt(q.Get("end"), 10, 64)

	f := store.TransactionFilter{
		AccountID:       q.Get("account_id"),
		Category:        q.Get("category"),
		Search:          q.Get("search"),
		StartDate:       startDate,
		EndDate:         endDate,
		IncludePositive: q.Get("include_positive") == "true",
		Page:            page,
		Limit:           limit,
		SortBy:          q.Get("sort_by"),
		SortDir:         q.Get("sort_dir"),
	}

	txns, total, err := h.store.List(r.Context(), f)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 50
	}

	WriteDataWithMeta(w, txns, &Meta{Page: f.Page, Limit: f.Limit, Total: total})
}

func (h *TransactionHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	txn, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if txn == nil {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "Transaction not found")
		return
	}
	WriteData(w, txn)
}

func (h *TransactionHandler) OverrideCategory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Category        string `json:"category"`
		ApplyToMerchant bool   `json:"apply_to_merchant"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body")
		return
	}

	if body.Category == "" {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Category is required")
		return
	}

	ctx := r.Context()

	if body.ApplyToMerchant {
		txn, err := h.store.GetByID(ctx, id)
		if err != nil || txn == nil {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "Transaction not found")
			return
		}
		if err := h.catStore.SetOverrideAndMerchant(ctx, id, txn.Description, body.Category); err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
	} else {
		if err := h.catStore.SetOverride(ctx, id, body.Category); err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
	}

	h.events.Broadcast("categories_updated", `{"status":"ok"}`)
	WriteData(w, map[string]string{"status": "ok"})
}

func (h *TransactionHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	startDate, _ := strconv.ParseInt(q.Get("start"), 10, 64)
	endDate, _ := strconv.ParseInt(q.Get("end"), 10, 64)

	f := store.TransactionFilter{
		AccountID:       q.Get("account_id"),
		Category:        q.Get("category"),
		Search:          q.Get("search"),
		StartDate:       startDate,
		EndDate:         endDate,
		IncludePositive: q.Get("include_positive") == "true",
		Page:            1,
		Limit:           10000,
		SortBy:          "posted",
		SortDir:         "desc",
	}

	txns, _, err := h.store.List(r.Context(), f)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=transactions.csv")

	cw := csv.NewWriter(w)
	cw.Write([]string{"Date", "Description", "Amount", "Category", "Account ID"})
	for _, t := range txns {
		cw.Write([]string{
			strconv.FormatInt(t.Posted, 10),
			t.Description,
			fmt.Sprintf("%.2f", t.Amount),
			t.Category,
			t.AccountID,
		})
	}
	cw.Flush()
}
