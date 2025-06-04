package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"finance_tracker/src/internal/auth"
	"finance_tracker/src/internal/usecases"
)

// TransactionHandlers handles transaction-related HTTP requests
type TransactionHandlers struct {
	transactionUseCase *usecases.TransactionUseCase
}

// NewTransactionHandlers creates new transaction handlers
func NewTransactionHandlers(transactionUseCase *usecases.TransactionUseCase) *TransactionHandlers {
	return &TransactionHandlers{
		transactionUseCase: transactionUseCase,
	}
}

// HandleGetTransactions returns paginated transactions for the current organization
func (h *TransactionHandlers) HandleGetTransactions(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	orgID := auth.GetOrganization(r.Context())
	if user == nil || orgID.String() == "00000000-0000-0000-0000-000000000000" {
		respondWithError(w, r, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	// Parse query parameters
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Parse optional filters
	req := usecases.ListTransactionsRequest{
		Limit:  limit,
		Offset: offset,
	}

	if search := r.URL.Query().Get("search"); search != "" {
		req.Search = &search
	}

	if category := r.URL.Query().Get("category"); category != "" {
		req.Category = &category
	}

	if accountIDStr := r.URL.Query().Get("account_id"); accountIDStr != "" {
		if accountID, err := uuid.Parse(accountIDStr); err == nil {
			req.AccountID = &accountID
		}
	}

	response, err := h.transactionUseCase.ListTransactions(r.Context(), orgID, req)
	if err != nil {
		respondWithError(w, r, http.StatusInternalServerError, "Failed to get transactions", err)
		return
	}

	respondWithJSON(w, r, http.StatusOK, response)
}

// HandleGetRecentTransactions returns recent transactions for dashboard
func (h *TransactionHandlers) HandleGetRecentTransactions(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	orgID := auth.GetOrganization(r.Context())
	if user == nil || orgID.String() == "00000000-0000-0000-0000-000000000000" {
		respondWithError(w, r, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	transactions, err := h.transactionUseCase.GetRecentTransactions(r.Context(), orgID, 10)
	if err != nil {
		respondWithError(w, r, http.StatusInternalServerError, "Failed to get recent transactions", err)
		return
	}

	respondWithJSON(w, r, http.StatusOK, transactions)
}

// HandleGetTransactionDetail returns detailed information about a specific transaction
func (h *TransactionHandlers) HandleGetTransactionDetail(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	orgID := auth.GetOrganization(r.Context())
	if user == nil || orgID.String() == "00000000-0000-0000-0000-000000000000" {
		respondWithError(w, r, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	transactionIDStr := chi.URLParam(r, "transactionID")
	if transactionIDStr == "" {
		respondWithError(w, r, http.StatusBadRequest, "Transaction ID is required", nil)
		return
	}

	transactionID, err := uuid.Parse(transactionIDStr)
	if err != nil {
		respondWithError(w, r, http.StatusBadRequest, "Invalid transaction ID", err)
		return
	}

	transaction, err := h.transactionUseCase.GetTransactionDetail(r.Context(), orgID, transactionID)
	if err != nil {
		respondWithError(w, r, http.StatusNotFound, "Transaction not found", err)
		return
	}

	respondWithJSON(w, r, http.StatusOK, transaction)
}

// HandleUpdateTransactionCategory updates the category of a transaction
func (h *TransactionHandlers) HandleUpdateTransactionCategory(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	orgID := auth.GetOrganization(r.Context())
	if user == nil || orgID.String() == "00000000-0000-0000-0000-000000000000" {
		respondWithError(w, r, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	transactionIDStr := chi.URLParam(r, "transactionID")
	if transactionIDStr == "" {
		respondWithError(w, r, http.StatusBadRequest, "Transaction ID is required", nil)
		return
	}

	transactionID, err := uuid.Parse(transactionIDStr)
	if err != nil {
		respondWithError(w, r, http.StatusBadRequest, "Invalid transaction ID", err)
		return
	}

	var reqBody struct {
		Category string `json:"category"`
	}
	if err := parseJSON(r, &reqBody); err != nil {
		respondWithError(w, r, http.StatusBadRequest, "Invalid request", err)
		return
	}
	category := reqBody.Category

	if category == "" {
		respondWithError(w, r, http.StatusBadRequest, "Category is required", nil)
		return
	}

	req := usecases.UpdateTransactionCategoryRequest{
		TransactionID: transactionID,
		Category:      category,
	}

	err = h.transactionUseCase.UpdateTransactionCategory(r.Context(), orgID, req)
	if err != nil {
		if valErr, ok := err.(*usecases.ValidationError); ok {
			respondWithError(w, r, http.StatusBadRequest, valErr.Message, err)
			return
		}
		respondWithError(w, r, http.StatusInternalServerError, "Failed to update transaction category", err)
		return
	}

	log.Info().
		Str("transaction_id", transactionID.String()).
		Str("new_category", category).
		Str("user_id", user.ID.String()).
		Msg("Updated transaction category")

	respondWithJSON(w, r, http.StatusOK, map[string]interface{}{
		"success":  true,
		"category": category,
	})
}
