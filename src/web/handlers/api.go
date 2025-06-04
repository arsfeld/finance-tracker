package handlers

import (
	"encoding/json"
	"net/http"

	"finance_tracker/src/internal/config"
)

type APIHandlers struct {
	client *config.Client
}

func NewAPIHandlers(client *config.Client) *APIHandlers {
	return &APIHandlers{client: client}
}

// Minimal handlers that return empty responses for now

func (h *APIHandlers) HandleGetTransactions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, []interface{}{})
}

func (h *APIHandlers) HandleGetRecentTransactions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, []interface{}{})
}

func (h *APIHandlers) HandleGetTransactionDetail(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{"message": "Not implemented"})
}

func (h *APIHandlers) HandleUpdateTransactionCategory(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{"message": "Not implemented"})
}

func (h *APIHandlers) HandleGetAccounts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, []interface{}{})
}

func (h *APIHandlers) HandleGetConnections(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, []interface{}{})
}

func (h *APIHandlers) HandleCreateConnection(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{"message": "Not implemented"})
}

func (h *APIHandlers) HandleDeleteConnection(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{"message": "Not implemented"})
}

func (h *APIHandlers) HandleTestConnection(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{"message": "Not implemented"})
}

func (h *APIHandlers) HandleGetConnectionAccounts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, []interface{}{})
}

func (h *APIHandlers) HandleUpdateAccountStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{"message": "Not implemented"})
}

func (h *APIHandlers) HandleGetDashboardStats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"total_accounts": 0,
		"total_transactions": 0,
		"current_balance": 0,
	})
}

// Helper function to write JSON response
func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}