package handlers

import (
	"net/http"

	"finance_tracker/src/internal/auth"
	"finance_tracker/src/internal/usecases"
)

// AccountHandlers handles account-related HTTP requests
type AccountHandlers struct {
	accountUseCase *usecases.AccountUseCase
}

// NewAccountHandlers creates new account handlers
func NewAccountHandlers(accountUseCase *usecases.AccountUseCase) *AccountHandlers {
	return &AccountHandlers{
		accountUseCase: accountUseCase,
	}
}

// HandleGetAccounts returns accounts for the current organization
func (h *AccountHandlers) HandleGetAccounts(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	orgID := auth.GetOrganization(r.Context())
	if user == nil || orgID.String() == "00000000-0000-0000-0000-000000000000" {
		respondWithError(w, r, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	accounts, err := h.accountUseCase.ListAccounts(r.Context(), orgID)
	if err != nil {
		respondWithError(w, r, http.StatusInternalServerError, "Failed to get accounts", err)
		return
	}

	respondWithJSON(w, r, http.StatusOK, accounts)
}
