package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"finance_tracker/src/internal/auth"
	"finance_tracker/src/internal/usecases"
)

// ConnectionHandlers handles connection-related HTTP requests
type ConnectionHandlers struct {
	connectionUseCase *usecases.ConnectionUseCase
}

// NewConnectionHandlers creates new connection handlers
func NewConnectionHandlers(connectionUseCase *usecases.ConnectionUseCase) *ConnectionHandlers {
	return &ConnectionHandlers{
		connectionUseCase: connectionUseCase,
	}
}

// HandleGetConnections returns all provider connections for the organization
func (h *ConnectionHandlers) HandleGetConnections(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	orgID := auth.GetOrganization(r.Context())
	if user == nil || orgID.String() == "00000000-0000-0000-0000-000000000000" {
		respondWithError(w, r, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	connections, err := h.connectionUseCase.ListConnections(r.Context(), orgID)
	if err != nil {
		respondWithError(w, r, http.StatusInternalServerError, "Failed to get connections", err)
		return
	}

	respondWithJSON(w, r, http.StatusOK, connections)
}

// HandleCreateConnection creates a new provider connection
func (h *ConnectionHandlers) HandleCreateConnection(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	orgID := auth.GetOrganization(r.Context())
	if user == nil || orgID.String() == "00000000-0000-0000-0000-000000000000" {
		respondWithError(w, r, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	var req usecases.CreateConnectionRequest
	if err := parseJSON(r, &req); err != nil {
		respondWithError(w, r, http.StatusBadRequest, "Invalid request", err)
		return
	}

	log.Debug().
		Str("provider_type", req.ProviderType).
		Str("connection_name", req.Name).
		Str("user_id", user.ID.String()).
		Str("org_id", orgID.String()).
		Msg("Creating new connection")

	connection, err := h.connectionUseCase.CreateConnection(r.Context(), orgID, req)
	if err != nil {
		if valErr, ok := err.(*usecases.ValidationError); ok {
			respondWithError(w, r, http.StatusBadRequest, valErr.Message, err)
			return
		}
		
		// Handle provider errors
		log.Error().
			Err(err).
			Str("provider_type", req.ProviderType).
			Str("connection_name", req.Name).
			Msg("Failed to create connection")
			
		respondWithError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	log.Info().
		Str("connection_id", connection.ID.String()).
		Str("provider_type", req.ProviderType).
		Str("connection_name", req.Name).
		Msg("Successfully created connection")

	respondWithJSON(w, r, http.StatusCreated, connection)
}

// HandleDeleteConnection deletes a provider connection
func (h *ConnectionHandlers) HandleDeleteConnection(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	orgID := auth.GetOrganization(r.Context())
	if user == nil || orgID.String() == "00000000-0000-0000-0000-000000000000" {
		respondWithError(w, r, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	connectionIDStr := chi.URLParam(r, "connectionID")
	if connectionIDStr == "" {
		respondWithError(w, r, http.StatusBadRequest, "Connection ID is required", nil)
		return
	}

	connectionID, err := uuid.Parse(connectionIDStr)
	if err != nil {
		respondWithError(w, r, http.StatusBadRequest, "Invalid connection ID", err)
		return
	}

	err = h.connectionUseCase.DeleteConnection(r.Context(), orgID, connectionID)
	if err != nil {
		respondWithError(w, r, http.StatusInternalServerError, "Failed to delete connection", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleTestConnection tests a provider connection
func (h *ConnectionHandlers) HandleTestConnection(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	orgID := auth.GetOrganization(r.Context())
	if user == nil || orgID.String() == "00000000-0000-0000-0000-000000000000" {
		respondWithError(w, r, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	connectionIDStr := chi.URLParam(r, "connectionID")
	if connectionIDStr == "" {
		respondWithError(w, r, http.StatusBadRequest, "Connection ID is required", nil)
		return
	}

	connectionID, err := uuid.Parse(connectionIDStr)
	if err != nil {
		respondWithError(w, r, http.StatusBadRequest, "Invalid connection ID", err)
		return
	}

	success, errorMessage, err := h.connectionUseCase.TestConnection(r.Context(), orgID, connectionID)
	if err != nil {
		respondWithError(w, r, http.StatusInternalServerError, "Failed to test connection", err)
		return
	}

	status := "success"
	if !success {
		status = "error"
	}

	response := map[string]interface{}{
		"success":       success,
		"status":        status,
		"error_message": errorMessage,
	}

	respondWithJSON(w, r, http.StatusOK, response)
}

// HandleGetConnectionAccounts returns all bank accounts for a connection
func (h *ConnectionHandlers) HandleGetConnectionAccounts(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	orgID := auth.GetOrganization(r.Context())
	if user == nil || orgID.String() == "00000000-0000-0000-0000-000000000000" {
		respondWithError(w, r, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	connectionIDStr := chi.URLParam(r, "connectionID")
	if connectionIDStr == "" {
		respondWithError(w, r, http.StatusBadRequest, "Connection ID is required", nil)
		return
	}

	connectionID, err := uuid.Parse(connectionIDStr)
	if err != nil {
		respondWithError(w, r, http.StatusBadRequest, "Invalid connection ID", err)
		return
	}

	accounts, err := h.connectionUseCase.ListConnectionAccounts(r.Context(), orgID, connectionID)
	if err != nil {
		respondWithError(w, r, http.StatusInternalServerError, "Failed to get connection accounts", err)
		return
	}

	respondWithJSON(w, r, http.StatusOK, accounts)
}

// HandleUpdateAccountStatus toggles the active status of a bank account
func (h *ConnectionHandlers) HandleUpdateAccountStatus(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	orgID := auth.GetOrganization(r.Context())
	if user == nil || orgID.String() == "00000000-0000-0000-0000-000000000000" {
		respondWithError(w, r, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	accountIDStr := chi.URLParam(r, "accountID")
	if accountIDStr == "" {
		respondWithError(w, r, http.StatusBadRequest, "Account ID is required", nil)
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		respondWithError(w, r, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	var req struct {
		IsActive bool `json:"is_active"`
	}

	if err := parseJSON(r, &req); err != nil {
		respondWithError(w, r, http.StatusBadRequest, "Invalid request", err)
		return
	}

	err = h.connectionUseCase.UpdateAccountStatus(r.Context(), orgID, accountID, req.IsActive)
	if err != nil {
		respondWithError(w, r, http.StatusInternalServerError, "Failed to update account status", err)
		return
	}

	respondWithJSON(w, r, http.StatusOK, map[string]interface{}{
		"success":   true,
		"is_active": req.IsActive,
	})
}