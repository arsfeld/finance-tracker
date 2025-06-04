package handlers

import (
	"net/http"

	"finance_tracker/src/internal/config"
	"finance_tracker/src/internal/services"
	"finance_tracker/src/internal/usecases"
)

// APIHandlers aggregates all API handlers
type APIHandlers struct {
	Transaction  *TransactionHandlers
	Connection   *ConnectionHandlers
	Account      *AccountHandlers
	Dashboard    *DashboardHandlers
}

// NewAPIHandlers creates a new API handlers instance with all dependencies
func NewAPIHandlers(client *config.Client) *APIHandlers {
	// Create services
	transactionService := services.NewTransactionService(client)
	connectionService := services.NewConnectionService(client)
	accountService := services.NewAccountService(client)
	providerService := services.NewProviderService()
	analyticsService := services.NewAnalyticsService(client)

	// Create use cases
	transactionUseCase := usecases.NewTransactionUseCase(transactionService)
	connectionUseCase := usecases.NewConnectionUseCase(connectionService, accountService, providerService)
	accountUseCase := usecases.NewAccountUseCase(accountService)
	dashboardUseCase := usecases.NewDashboardUseCase(analyticsService)

	// Create handlers
	return &APIHandlers{
		Transaction: NewTransactionHandlers(transactionUseCase),
		Connection:  NewConnectionHandlers(connectionUseCase),
		Account:     NewAccountHandlers(accountUseCase),
		Dashboard:   NewDashboardHandlers(dashboardUseCase),
	}
}

// Legacy handler methods for backward compatibility
// These delegate to the new structured handlers

func (h *APIHandlers) HandleGetTransactions(w http.ResponseWriter, r *http.Request) {
	h.Transaction.HandleGetTransactions(w, r)
}

func (h *APIHandlers) HandleGetRecentTransactions(w http.ResponseWriter, r *http.Request) {
	h.Transaction.HandleGetRecentTransactions(w, r)
}

func (h *APIHandlers) HandleGetTransactionDetail(w http.ResponseWriter, r *http.Request) {
	h.Transaction.HandleGetTransactionDetail(w, r)
}

func (h *APIHandlers) HandleUpdateTransactionCategory(w http.ResponseWriter, r *http.Request) {
	h.Transaction.HandleUpdateTransactionCategory(w, r)
}

func (h *APIHandlers) HandleGetAccounts(w http.ResponseWriter, r *http.Request) {
	h.Account.HandleGetAccounts(w, r)
}

func (h *APIHandlers) HandleGetConnections(w http.ResponseWriter, r *http.Request) {
	h.Connection.HandleGetConnections(w, r)
}

func (h *APIHandlers) HandleCreateConnection(w http.ResponseWriter, r *http.Request) {
	h.Connection.HandleCreateConnection(w, r)
}

func (h *APIHandlers) HandleDeleteConnection(w http.ResponseWriter, r *http.Request) {
	h.Connection.HandleDeleteConnection(w, r)
}

func (h *APIHandlers) HandleTestConnection(w http.ResponseWriter, r *http.Request) {
	h.Connection.HandleTestConnection(w, r)
}

func (h *APIHandlers) HandleGetConnectionAccounts(w http.ResponseWriter, r *http.Request) {
	h.Connection.HandleGetConnectionAccounts(w, r)
}

func (h *APIHandlers) HandleUpdateAccountStatus(w http.ResponseWriter, r *http.Request) {
	h.Connection.HandleUpdateAccountStatus(w, r)
}

func (h *APIHandlers) HandleGetDashboardStats(w http.ResponseWriter, r *http.Request) {
	h.Dashboard.HandleGetDashboardStats(w, r)
}